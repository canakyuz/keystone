// Command worker is the separate process that runs provisioning jobs.
//
// WHY a separate process from the Control API:
//
//   - Provisioning takes a long time. Running it in the API process would make
//     request-serving capacity compete with provisioning load.
//   - It needs different database privileges. The worker creates schemas; the API
//     does not. A separate process makes splitting those privileges possible.
//     (That split has not been done yet, see docs/decisions/0001.)
//   - It needs a concurrency limit independent of user requests. In one process the
//     two limits would interfere.
//
// Scaling: several worker instances can run at once. Because claiming uses FOR
// UPDATE SKIP LOCKED, the same job is never taken twice (see docs/decisions/0002).
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	"github.com/canakyuz/keystone/internal/config"
	operationRepo "github.com/canakyuz/keystone/internal/repository/operation"
	templateRepo "github.com/canakyuz/keystone/internal/repository/template"
	tenantRepo "github.com/canakyuz/keystone/internal/repository/tenant"
	tenantUsecase "github.com/canakyuz/keystone/internal/usecase/tenant"
	"github.com/canakyuz/keystone/internal/worker"
	"github.com/canakyuz/keystone/pkg/database"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/metrics"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "worker durdu: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("could not load configuration: %w", err)
	}

	log := logger.New(logger.Config{
		Level:       cfg.Server.Environment,
		Environment: cfg.Server.Environment,
	})

	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("could not connect to the database: %w", err)
	}
	defer db.Close()

	// The worker identity is recorded as the lease owner. The hostname is used so that
	// with several instances running it is visible which job is where.
	workerID := workerIdentity()

	operations := operationRepo.New(db)
	tenants := tenantRepo.NewPostgresRepository(db)
	provisioner := tenantUsecase.NewProvisioningService(db, templateRepo.NewFileSystemRepository("templates/tenants"), log)

	handler := worker.NewProvisionHandler(tenants, tenants, provisioner, log)

	// The worker publishes its own metrics on its own listener. It is a separate
	// process from the API, so it needs a separate scrape target; sharing the API's
	// endpoint would attribute the worker's numbers to the API and would stop working
	// the moment the two are scaled independently, which is the point of splitting them.
	metricsRegistry := metrics.New()
	stopMetrics := serveMetrics(cfg.Server.MetricsAddr, metricsRegistry, log)
	defer stopMetrics()

	cfgWorker := worker.DefaultConfig(workerID)
	cfgWorker.Metrics = metricsRegistry
	provisionerWorker := worker.New(cfgWorker, operations, handler.Handle, log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.WithFields(logger.Fields{
		"worker_id":      workerID,
		"max_concurrent": cfgWorker.MaxConcurrent,
		"max_per_tenant": cfgWorker.MaxPerTenant,
		"lease_duration": cfgWorker.LeaseDuration.String(),
		"shutdown_grace": cfgWorker.ShutdownGrace.String(),
	}).Info("worker started")

	// Run continues until ctx is cancelled. Shutdown order: claiming stops, running
	// jobs get a bounded grace period, and are cancelled when it expires. Jobs that
	// cannot finish are taken over once their lease expires.
	err = provisionerWorker.Run(ctx)

	log.Info("worker stopped")

	return err
}

// workerIdentity builds this instance's identity.
func workerIdentity() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}

	return fmt.Sprintf("%s-%d", host, os.Getpid())
}
