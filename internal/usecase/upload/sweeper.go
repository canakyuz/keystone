// Package upload holds the work on stored files that no single request does.
package upload

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/canakyuz/keystone/pkg/logger"
)

// SweepGrace is how old a file with no record must be before a sweep counts it.
//
// A file reaches disk a moment before its record commits, and a sweep that ran in between
// would see a file with no row. An hour is far longer than that moment.
const SweepGrace = time.Hour

// SweepInterval is how often the API process sweeps.
const SweepInterval = time.Hour

// PathLister reports the served paths recorded for one tenant's files.
type PathLister interface {
	Paths(ctx context.Context, tenantID string) (map[string]struct{}, error)
}

// SweepResult is what one sweep found.
type SweepResult struct {
	// Unrecorded holds the served paths of files older than SweepGrace that their tenant has
	// no record of.
	Unrecorded []string

	// Removed counts the files deleted, which is zero unless removal is on.
	Removed int
}

// Sweeper finds stored files that have no record: the ones left behind when the process
// died between writing a file and committing its row.
//
// It runs in the API process, because that is the process whose disk holds the files. The
// worker may run on another host.
//
// By default it only reports. Files stored before migration 041 have no record either, and
// those are a deployment's real logos and images; deleting every unrecorded file on the
// first sweep after an upgrade would delete them. An operator turns removal on with
// UPLOAD_SWEEP_REMOVE after reading the report.
type Sweeper struct {
	root   string
	lister PathLister
	remove bool
	log    *logger.Logger
	now    func() time.Time
}

// NewSweeper creates a sweeper over the upload root the handler writes to.
func NewSweeper(root string, lister PathLister, remove bool, log *logger.Logger) *Sweeper {
	return &Sweeper{root: root, lister: lister, remove: remove, log: log, now: time.Now}
}

// Run sweeps once at once and then every SweepInterval, until ctx is cancelled.
func (s *Sweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(SweepInterval)
	defer ticker.Stop()

	for {
		s.report(s.Sweep(ctx))

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Sweep walks the tenant directories once.
//
// Time O(F + R): each file is visited once, and each tenant's recorded paths are read once,
// in one query, into a set. Space O(R) for the tenant with the most records.
func (s *Sweeper) Sweep(ctx context.Context) (SweepResult, error) {
	var result SweepResult

	tenantsDir := filepath.Join(s.root, "tenants")
	tenants, err := os.ReadDir(tenantsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("could not list the upload directory: %w", err)
	}

	for _, tenant := range tenants {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		// Only a directory named by a tenant id is the handler's. Anything else is not this
		// sweep's to judge, and its name is not safe to hand to the database as a tenant.
		if !tenant.IsDir() || len(tenant.Name()) != 36 || uuid.Validate(tenant.Name()) != nil {
			continue
		}

		if err := s.sweepTenant(ctx, tenantsDir, tenant.Name(), &result); err != nil {
			return result, err
		}
	}

	return result, nil
}

// sweepTenant checks one tenant's files against that tenant's own records. A row in another
// tenant never accounts for a file here, because the records are read under this tenant's
// scope.
func (s *Sweeper) sweepTenant(ctx context.Context, tenantsDir, tenantID string, result *SweepResult) error {
	recorded, err := s.lister.Paths(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("could not read the upload records of tenant %s: %w", tenantID, err)
	}

	cutoff := s.now().Add(-SweepGrace)

	return filepath.WalkDir(filepath.Join(tenantsDir, tenantID), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.Type().IsRegular() {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(cutoff) {
			return nil
		}

		relative, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		served := "/uploads/" + filepath.ToSlash(relative)
		if _, ok := recorded[served]; ok {
			return nil
		}

		result.Unrecorded = append(result.Unrecorded, served)
		if !s.remove {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("could not remove an unrecorded upload: %w", err)
		}
		result.Removed++
		return nil
	})
}

// report logs what a sweep found. A sweep that found nothing says nothing.
func (s *Sweeper) report(result SweepResult, err error) {
	if s.log == nil {
		return
	}

	if err != nil && !errors.Is(err, context.Canceled) {
		s.log.ErrorWithErr(err, "upload sweep failed")
	}

	if len(result.Unrecorded) == 0 {
		return
	}

	sample := result.Unrecorded[:min(len(result.Unrecorded), 10)]
	s.log.WithFields(logger.Fields{
		"unrecorded": len(result.Unrecorded),
		"removed":    result.Removed,
		"sample":     sample,
	}).Warn("stored files with no upload record")
}
