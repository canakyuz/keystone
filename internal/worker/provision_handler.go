package worker

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	"github.com/canakyuz/keystone/internal/domain/tenant"
	"github.com/canakyuz/keystone/pkg/logger"
)

// TenantLoader, isin ait oldugu tenant'i getirir.
// Arayuz tuketen tarafta tanimlanir: worker yalnizca ihtiyac duydugu tek
// davranisi bilir, tenant repository'sinin tamamini degil.
type TenantLoader interface {
	GetByID(ctx context.Context, id string) (*tenant.Tenant, error)
}

// SchemaProvisioner, tenant'in calisma alanini hazirlar.
type SchemaProvisioner interface {
	ProvisionTenantSchema(ctx context.Context, t *tenant.Tenant) error
}

// StatusSetter, tenant yasam dongusu durumunu gunceller.
//
// Metotlar bilincli olarak dar: "herhangi bir duruma gec" degil, "su gecisi
// yap". Gerekce, ADR-0003'te yazili sinirla ilgili. Fencing yalnizca is
// tablosundaki metadata guncellemesini korur; buradaki tenant yazimi o
// korumanin disindadir. Lease'ini kaybetmis eski bir worker bu cagriyi
// yapabilir.
//
// Bu yuzden gecisler kaynak duruma kosulludur. Aktif bir tenant'i geri
// 'provisioning' durumuna cekmek mumkun degildir; eski worker'in gecikmis
// cagrisi sessizce etkisiz kalir.
type StatusSetter interface {
	// MarkProvisioning, tenant'i 'provisioning' durumuna alir.
	// Tenant zaten 'active' ise hicbir sey yapmaz.
	MarkProvisioning(ctx context.Context, tenantID string) error

	// MarkFailed, tenant'i 'failed' durumuna alir.
	// Tenant zaten 'active' ise hicbir sey yapmaz.
	MarkFailed(ctx context.Context, tenantID string) error
}

// ProvisionHandler, kurulum isini yurutur.
//
// TEKRARLANABILIRLIK
// Bu handler birden fazla kez calisabilir ve calismalidir. Lease suresi dolan
// bir is baska bir worker'a gecer ve adimlar bastan calisir. Belgedeki arıza
// matrisinde bu satir soyle: "Sema olusturuldu, sonuc kaydedilmedi -> yeni
// deneme mevcut semayi dogrulayarak ilerler."
//
// Adimlarin her biri bu yuzden ya idempotent, ya da mevcut durumu dogrulayip
// gecen bicimde yazilmistir.
type ProvisionHandler struct {
	tenants     TenantLoader
	status      StatusSetter
	provisioner SchemaProvisioner
	log         *logger.Logger
}

// NewProvisionHandler, handler olusturur.
func NewProvisionHandler(
	tenants TenantLoader, status StatusSetter, provisioner SchemaProvisioner, log *logger.Logger,
) *ProvisionHandler {
	return &ProvisionHandler{tenants: tenants, status: status, provisioner: provisioner, log: log}
}

// Handle, tek bir kurulum isini yurutur.
//
// Adimlar:
//  1. Tenant kaydini yukle.
//  2. Zaten aktifse hicbir sey yapma (onceki denemenin sonucu kaydedilmemis).
//  3. Durumu 'provisioning' yap.
//  4. Semayi olustur ve migration'lari uygula.
//
// Tenant'i 'active' yapmak bu handler'in isi DEGILDIR. Aktiflestirme, isin
// tamamlandi olarak isaretlenmesiyle ayni transaction icinde yapilir
// (bkz. Repository.CompleteSuccess). Burada yapilsaydi, aktiflestirme ile
// operasyon kapatma arasinda surec kapandiginda kullaniciya "islemde" gorunen
// ama fiilen kullanilabilir bir tenant kalirdi.
func (h *ProvisionHandler) Handle(ctx context.Context, job *domain.Job) error {
	t, err := h.tenants.GetByID(ctx, job.TenantID)
	if err != nil {
		return fmt.Errorf("tenant yüklenemedi: %w", err)
	}
	if t == nil {
		return errors.New("tenant bulunamadı")
	}

	// Onceki deneme isi bitirmis ama sonucunu kaydedememis olabilir.
	// Bu durumda adimlari tekrar calistirmaya gerek yok.
	if t.Status == tenant.TenantStatusActive {
		h.logStep(job, "tenant zaten aktif, kurulum atlandı")
		return nil
	}

	if err := h.status.MarkProvisioning(ctx, t.ID); err != nil {
		return fmt.Errorf("durum 'provisioning' yapılamadı: %w", err)
	}

	h.logStep(job, "şema hazırlanıyor")

	// ProvisionTenantSchema, CREATE SCHEMA IF NOT EXISTS kullanir ve
	// migration'lari kendi transaction'inda uygular. Yarida kesilirse
	// transaction geri alinir; yeni deneme temiz bir noktadan baslar.
	if err := h.provisioner.ProvisionTenantSchema(ctx, t); err != nil {
		return fmt.Errorf("şema hazırlanamadı: %w", err)
	}

	h.logStep(job, "şema hazır")

	return nil
}

// MarkFailed, deneme hakki tukendiginde tenant'i 'failed' isaretler.
//
// Ayri bir metot olmasinin nedeni: her basarisiz deneme tenant'i 'failed'
// yapmamali. Yeniden denenecek bir is icin tenant 'provisioning' kalir.
// Yalnizca hak tukendiginde son duruma gecilir.
func (h *ProvisionHandler) MarkFailed(ctx context.Context, tenantID string) error {
	return h.status.MarkFailed(ctx, tenantID)
}

// logStep, teshis zincirini tasir.
//
// Belgedeki zincir: request_id -> operation_id -> job_id -> attempt ->
// worker -> migration adimi. Burada operation_id, job_id ve attempt yazilir;
// request_id henuz HTTP katmanindan tasinmiyor.
func (h *ProvisionHandler) logStep(job *domain.Job, step string) {
	if h.log == nil {
		return
	}

	h.log.WithFields(logger.Fields{
		"operation_id": job.OperationID,
		"job_id":       job.ID,
		"tenant_id":    job.TenantID,
		"attempt":      job.Attempts,
		"lease_owner":  job.LeaseOwner,
		"fence":        job.Fence,
	}).Info(step)
}
