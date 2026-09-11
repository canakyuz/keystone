// Command worker is the separate process that runs provisioning jobs.
//
// WHY a separate process from the Control API:
//
//   - Provisioning takes a long time. Running it in the API process would make
//     request-serving capacity compete with provisioning load.
//   - It needs different database privileges. The worker creates schemas and crosses
//     tenants on the queue tables; the API does neither, and the worker reads no tenant
//     data. It connects as its own role, see migration 039 and docs/decisions/0001.
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
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/canakyuz/keystone/internal/config"
	auditRepo "github.com/canakyuz/keystone/internal/repository/audit"
	operationRepo "github.com/canakyuz/keystone/internal/repository/operation"
	outboxRepo "github.com/canakyuz/keystone/internal/repository/outbox"
	templateRepo "github.com/canakyuz/keystone/internal/repository/template"
	tenantRepo "github.com/canakyuz/keystone/internal/repository/tenant"
	tenantUsecase "github.com/canakyuz/keystone/internal/usecase/tenant"
	"github.com/canakyuz/keystone/internal/worker"
	"github.com/canakyuz/keystone/pkg/database"
	"github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/metrics"
	"github.com/canakyuz/keystone/pkg/tracing"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "worker stopped: %v\n", err)
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

	// Its own role, not the API's. See Config.WorkerDatabase.
	dbCfg, err := cfg.WorkerDatabase()
	if err != nil {
		return err
	}

	db, err := database.NewPostgresDB(dbCfg)
	if err != nil {
		return fmt.Errorf("could not connect to the database: %w", err)
	}
	defer db.Close()

	// The worker identity is recorded as the lease owner. The hostname is used so that
	// with several instances running it is visible which job is where.
	workerID := workerIdentity()

	outbox := outboxRepo.New(db)

	// The operations repository emits notifications in the transaction that completes a
	// job; see internal/repository/operation. Delivery is a separate loop below.
	operations := operationRepo.New(db).WithOutbox(outbox).WithAudit(auditRepo.New())
	tenants := tenantRepo.NewPostgresRepository(db)
	provisioner := tenantUsecase.NewProvisioningService(db, templateRepo.NewFileSystemRepository("templates/tenants"), log)

	handler := worker.NewProvisionHandler(tenants, tenants, provisioner, log)

	// The worker publishes its own metrics on its own listener. It is a separate
	// process from the API, so it needs a separate scrape target; sharing the API's
	// endpoint would attribute the worker's numbers to the API and would stop working
	// the moment the two are scaled independently, which is the point of splitting them.
	// A separate service name from the API. They are separate processes, and attributing
	// the worker's spans to the API would make the two indistinguishable in the collector.
	shutdownTracing, err := tracing.Init(context.Background(), tracing.Config{
		Endpoint:    cfg.Tracing.Endpoint,
		ServiceName: "keystone-worker",
		Environment: cfg.Server.Environment,
		SampleRatio: cfg.Tracing.SampleRatio,
	})
	if err != nil {
		return fmt.Errorf("could not initialise tracing: %w", err)
	}
	defer func() {
		// Spans leave in batches, so without an explicit flush the last few seconds of a
		// run are lost, which is the window that matters when a worker is being restarted.
		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTracing(flushCtx)
	}()

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

	// The delivery loop runs beside the provisioning loop in this process.
	//
	// Beside, not inside: they claim from different tables at different rates, and a
	// backlog of webhook retries must not delay a tenant being provisioned. Splitting
	// them into separate processes later needs no code change, only a flag — which is the
	// reason they are separate loops rather than one.
	outboxCfg := worker.DefaultOutboxConfig(workerID)
	outboxCfg.Metrics = metricsRegistry
	outboxWorker := worker.NewOutbox(outboxCfg, outbox, nil, log)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := outboxWorker.Run(ctx); err != nil {
			log.WithFields(logger.Fields{"error": err.Error()}).Warn("outbox worker stopped")
		}
	}()

	// Run continues until ctx is cancelled. Shutdown order: claiming stops, running
	// jobs get a bounded grace period, and are cancelled when it expires. Jobs that
	// cannot finish are taken over once their lease expires.
	err = provisionerWorker.Run(ctx)

	wg.Wait()

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
