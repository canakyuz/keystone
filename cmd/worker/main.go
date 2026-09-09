// Command worker, kurulum islerini yuruten ayri surectir.
//
// NEDEN Control API'den ayri bir surec:
//
//   - Kurulum isleri uzun surer. API surecinde calistirmak, istek isleme
//     kapasitesini kurulum yuku ile paylastirirdi.
//   - Farkli veritabani yetkileri gerektirir. Worker sema olusturur; API
//     olusturmaz. Ayri surec, yetkilerin ayrilmasini mumkun kilar.
//     (Bu ayrim henuz yapilmadi, bkz. docs/decisions/0001.)
//   - Kullanici isteklerinden bagimsiz bir eszamanlilik sinirina ihtiyac
//     duyar. Ayni surecte olsalardi iki sinir birbirine karisirdi.
//
// Olceklendirme: birden fazla worker ornegi ayni anda calisabilir. Is
// sahiplenme FOR UPDATE SKIP LOCKED ile yapildigi icin ayni is iki kez
// alinmaz (bkz. docs/decisions/0002).
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

	// Worker kimligi lease sahipligine yazilir. Hostname kullanilir ki
	// birden fazla ornek calistiginda hangi isin nerede oldugu gorulebilsin.
	workerID := workerIdentity()

	operations := operationRepo.New(db)
	tenants := tenantRepo.NewPostgresRepository(db)
	provisioner := tenantUsecase.NewProvisioningService(db, templateRepo.NewFileSystemRepository("templates/tenants"), log)

	handler := worker.NewProvisionHandler(tenants, tenants, provisioner, log)

	cfgWorker := worker.DefaultConfig(workerID)
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

	// Run, ctx iptal edilene kadar surer. Kapanma sirasi:
	// yeni is alimi durur, calisanlara sinirli sure verilir, sure dolunca
	// iptal edilir. Tamamlanamayan isler lease suresi dolunca devralinir.
	err = provisionerWorker.Run(ctx)

	log.Info("Worker durdu")

	return err
}

// workerIdentity, bu ornegin kimligini uretir.
func workerIdentity() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}

	return fmt.Sprintf("%s-%d", host, os.Getpid())
}
