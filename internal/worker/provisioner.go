// Package worker, kurulum islerini yurutur.
//
// Tasarim kararlari ve gerekceleri:
//
//   - Is basina sinirsiz goroutine acilmaz. Kaynaklari korumadan ilerlemek
//     kolaydir; sistemin altinda kalmadan ilerlemesi tasarim gerektirir.
//   - Eszamanlilik hem toplamda hem tenant basina sinirlanir. Tenant siniri
//     olmadan tek bir musterinin yuku butun kapasiteyi tuketebilir.
//   - Kapanma sirasinda once yeni is alimi durur, calisanlara sinirli sure
//     verilir, sure dolunca iptal edilir. Tamamlanamayan isler lease suresi
//     dolunca baska bir worker tarafindan devralinir.
//   - context iptali, daha once commit edilmis yan etkileri geri ALMAZ.
//     Bu ayrim korunur: iptal "durmayi dene" demektir, "yapilani sil" demez.
package worker

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
	"github.com/canakyuz/keystone/pkg/logger"
)

// Handler, bir isin gercek islemini yurutur.
//
// Sozlesme: guvenle tekrar calistirilabilir olmalidir. Lease suresi dolan bir
// is baska bir worker'a gecebilir ve ayni adimlar bastan calisir. "En fazla
// bir kez calisir" garantisi verilmez.
type Handler func(ctx context.Context, job *domain.Job) error

// Config, worker'in sinirlarini belirler.
type Config struct {
	// ID, bu worker surecinin kimligidir. Lease sahipligi bununla yazilir.
	ID string

	// MaxConcurrent, ayni anda yurutulen toplam is sayisidir.
	MaxConcurrent int

	// MaxPerTenant, tek bir tenant icin ayni anda yurutulen is sayisidir.
	//
	// Neden ayri sinir: yalnizca toplam sinir olsaydi, cok isi olan bir
	// tenant butun yuvalari doldurup digerlerini bekletebilirdi.
	MaxPerTenant int

	// LeaseDuration, sahiplenmenin suresidir. Bir isin tipik suresinden
	// belirgin olarak uzun olmalidir, aksi halde is bitmeden lease duser.
	LeaseDuration time.Duration

	// LeaseRenewInterval, lease yenileme sikligidir.
	// LeaseDuration'un ucte birinden kucuk olmalidir: bir yenileme kacirilsa
	// bile lease dusmeden ikinci deneme yapilabilsin.
	LeaseRenewInterval time.Duration

	// PollInterval, is bulunamadiginda beklenecek suredir.
	PollInterval time.Duration

	// JobTimeout, tek bir isin ust sinirdir.
	JobTimeout time.Duration

	// ShutdownGrace, kapanirken calisan islere verilen suredir.
	ShutdownGrace time.Duration

	Backoff domain.BackoffConfig
}

// DefaultConfig, makul varsayilanlari dondurur.
func DefaultConfig(id string) Config {
	return Config{
		ID:                 id,
		MaxConcurrent:      8,
		MaxPerTenant:       2,
		LeaseDuration:      60 * time.Second,
		LeaseRenewInterval: 15 * time.Second,
		PollInterval:       time.Second,
		JobTimeout:         5 * time.Minute,
		ShutdownGrace:      30 * time.Second,
		Backoff:            domain.DefaultBackoff(),
	}
}

// Store, worker'in ihtiyac duydugu depolama davranislarini tanimlar.
//
// Arayuz burada, tuketen tarafta tanimlanir: worker yalnizca kullandigi dort
// metodu bilir, repository'nin tamamini degil.
type Store interface {
	Claim(ctx context.Context, workerID string, lease time.Duration) (*domain.Job, error)
	RenewLease(ctx context.Context, jobID, workerID string, fence int64, lease time.Duration) error
	CompleteSuccess(ctx context.Context, jobID, workerID string, fence int64, activateTenant bool) error
	CompleteFailure(ctx context.Context, jobID, workerID string, fence int64,
		errCode, errMessage string, retryAfter time.Duration) error
}

// Provisioner, kurulum isleri icin worker dongusudur.
type Provisioner struct {
	cfg     Config
	store   Store
	handler Handler
	log     *logger.Logger

	// slots, toplam eszamanlilik sinirini uygular.
	slots chan struct{}

	// tenantGate, tenant basina eszamanlilik sinirini uygular.
	tenantMu sync.Mutex
	tenant   map[string]int

	wg sync.WaitGroup
}

// New, worker olusturur.
func New(cfg Config, store Store, handler Handler, log *logger.Logger) *Provisioner {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 1
	}
	if cfg.MaxPerTenant <= 0 {
		cfg.MaxPerTenant = 1
	}

	return &Provisioner{
		cfg:     cfg,
		store:   store,
		handler: handler,
		log:     log,
		slots:   make(chan struct{}, cfg.MaxConcurrent),
		tenant:  make(map[string]int),
	}
}

// Run, worker dongusunu calistirir ve ctx iptal edilene kadar surer.
//
// Kapanma sirasi:
//  1. Yeni is alimi durur (dongu ctx.Done ile cikar).
//  2. Calisan islere ShutdownGrace kadar sure verilir.
//  3. Sure dolarsa is context'leri iptal edilir.
//  4. Tamamlanamayan isler lease suresi dolunca devralinir.
func (p *Provisioner) Run(ctx context.Context) error {
	// jobCtx, is context'lerinin ebeveynidir. Run'in ctx'inden AYRI tutulur:
	// boylece kapanma sinyali geldiginde calisan isleri hemen iptal etmeyip
	// once nazik sureyi taniyabiliriz.
	jobCtx, cancelJobs := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelJobs()

	p.loop(ctx, jobCtx)

	return p.drain(cancelJobs)
}

// loop, is bulup calistirmayi tekrarlar.
func (p *Provisioner) loop(ctx, jobCtx context.Context) {
	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Yuva bekle. Yuva yoksa yeni is talep etmeyiz; bos yere claim edip
		// isi elimizde tutmak, baska bir worker'in alabilecegi isi bloke eder.
		select {
		case <-ctx.Done():
			return
		case p.slots <- struct{}{}:
		}

		job, err := p.store.Claim(ctx, p.cfg.ID, p.cfg.LeaseDuration)
		if err != nil {
			<-p.slots
			if p.waitBeforeRetry(ctx, ticker, err) {
				return
			}
			continue
		}

		if !p.reserveTenant(job.TenantID) {
			// Bu tenant'in kotasi dolu. Isi hemen birakiyoruz; lease suresi
			// dolunca baska bir worker devralabilir. Elimizde tutmak, kotasi
			// musait olmayan bir isi kuyrukta kilitlerdi.
			p.releaseJob(jobCtx, job)
			<-p.slots
			continue
		}

		p.wg.Add(1)
		go p.execute(jobCtx, job)
	}
}

// waitBeforeRetry, is bulunamadiginda veya hata alindiginda bekler.
// Iptal edildiyse true doner.
func (p *Provisioner) waitBeforeRetry(ctx context.Context, ticker *time.Ticker, err error) bool {
	if !errors.Is(err, oprepo.ErrNoJob) && p.log != nil {
		p.log.WithFields(logger.Fields{"error": err.Error()}).Warn("İş sahiplenilemedi")
	}

	select {
	case <-ctx.Done():
		return true
	case <-ticker.C:
		return false
	}
}

// execute, tek bir isi yurutur ve sonucu bildirir.
func (p *Provisioner) execute(parent context.Context, job *domain.Job) {
	defer p.wg.Done()
	defer func() { <-p.slots }()
	defer p.releaseTenant(job.TenantID)

	ctx, cancel := context.WithTimeout(parent, p.cfg.JobTimeout)
	defer cancel()

	// Lease yenileme, is calisirken arka planda surer. Yenilenmezse uzun
	// suren bir is, henuz calisiyorken baska bir worker'a devredilirdi.
	stopRenew := p.startLeaseRenewal(ctx, job)
	defer stopRenew()

	// Panik, worker surecini dusurmemeli. Panige duşen is basarisiz sayilir
	// ve yeniden denenir.
	err := p.runHandler(ctx, job)

	p.report(job, err)
}

// runHandler, handler'i panik korumasiyla calistirir.
func (p *Provisioner) runHandler(ctx context.Context, job *domain.Job) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("iş panikledi: %v", recovered)
		}
	}()

	return p.handler(ctx, job)
}

// report, sonucu guncel fence ile bildirir.
//
// Bildirim icin ayri ve iptal edilemeyen bir context kullanilir: isin
// context'i zaman asimina ugramis olabilir, ama sonucun kaydedilmesi
// gerekir. Aksi halde is calisti, yan etkisi olustu, ama durumu
// guncellenmedigi icin bastan calistirilirdi.
func (p *Provisioner) report(job *domain.Job, runErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if runErr == nil {
		if err := p.store.CompleteSuccess(ctx, job.ID, p.cfg.ID, job.Fence, true); err != nil {
			p.logReportFailure(job, err)
		}
		return
	}

	retryAfter := domain.BackoffFor(p.cfg.Backoff, job.Attempts, rand.Float64())

	err := p.store.CompleteFailure(ctx, job.ID, p.cfg.ID, job.Fence,
		"provisioning_failed", runErr.Error(), retryAfter)
	if err != nil {
		p.logReportFailure(job, err)
	}
}

func (p *Provisioner) logReportFailure(job *domain.Job, err error) {
	if p.log == nil {
		return
	}

	// Fence eskimesi beklenen bir durumdur: is baska bir worker'a gecmistir.
	// Hata degil, bilgi olarak kaydedilir.
	level := p.log.WithFields(logger.Fields{
		"job_id":       job.ID,
		"operation_id": job.OperationID,
		"fence":        job.Fence,
		"error":        err.Error(),
	})

	if errors.Is(err, domain.ErrStaleFence) {
		level.Info("Sonuç bildirilemedi: iş devredilmiş")
		return
	}

	level.Error("Sonuç bildirilemedi")
}

// startLeaseRenewal, arka planda lease yenilemeyi baslatir ve durdurma
// fonksiyonu dondurur.
func (p *Provisioner) startLeaseRenewal(ctx context.Context, job *domain.Job) func() {
	renewCtx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(p.cfg.LeaseRenewInterval)
		defer ticker.Stop()

		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				err := p.store.RenewLease(renewCtx, job.ID, p.cfg.ID, job.Fence, p.cfg.LeaseDuration)
				if err != nil && p.log != nil {
					p.log.WithFields(logger.Fields{
						"job_id": job.ID,
						"error":  err.Error(),
					}).Warn("Lease yenilenemedi")
				}
			}
		}
	}()

	return cancel
}

// releaseJob, sahiplenilen ama calistirilamayan isi geri birakir.
func (p *Provisioner) releaseJob(ctx context.Context, job *domain.Job) {
	err := p.store.CompleteFailure(ctx, job.ID, p.cfg.ID, job.Fence,
		"tenant_capacity", "tenant eşzamanlılık sınırı dolu", p.cfg.PollInterval)
	if err != nil && p.log != nil {
		p.log.WithFields(logger.Fields{
			"job_id": job.ID,
			"error":  err.Error(),
		}).Warn("İş geri bırakılamadı")
	}
}

// reserveTenant, tenant kotasindan bir yuva ayirir.
// Karmasiklik: O(1).
func (p *Provisioner) reserveTenant(tenantID string) bool {
	p.tenantMu.Lock()
	defer p.tenantMu.Unlock()

	if p.tenant[tenantID] >= p.cfg.MaxPerTenant {
		return false
	}

	p.tenant[tenantID]++
	return true
}

// releaseTenant, tenant kotasindaki yuvayi birakir.
func (p *Provisioner) releaseTenant(tenantID string) {
	p.tenantMu.Lock()
	defer p.tenantMu.Unlock()

	p.tenant[tenantID]--
	if p.tenant[tenantID] <= 0 {
		// Haritanin sinirsiz buyumesini engelle.
		delete(p.tenant, tenantID)
	}
}

// drain, calisan islerin bitmesini bekler; sure dolunca iptal eder.
func (p *Provisioner) drain(cancelJobs context.CancelFunc) error {
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(p.cfg.ShutdownGrace):
		cancelJobs()
	}

	<-done

	// Iptal edilen isler tamamlanmis sayilmaz. Lease sureleri dolunca baska
	// bir worker tarafindan devralinirlar.
	return fmt.Errorf("kapanma süresi doldu, çalışan işler iptal edildi")
}
