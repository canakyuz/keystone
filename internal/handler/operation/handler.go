// Package operation, uzun suren islemlerin HTTP yuzunu saglar.
//
// Kurulum senkron degildir: istek isi kabul eder ve bir operasyon adresi
// dondurur. Istemci durumu o adresten sorgular.
//
// Neden senkron degil: sema olusturma ve migration uygulama saniyeler surebilir.
// Istegi bekletmek, zaman asimi durumunda istemciye isin ne olduguna dair
// hicbir bilgi birakmaz.
package operation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
	"github.com/canakyuz/keystone/pkg/logger"
)

// maxIdempotencyKeyLength, anahtarin ust sinirdir.
// Sema 255 karakter tutuyor; sinir burada da uygulanir ki veritabani hatasi
// yerine anlasilir bir dogrulama hatasi donsun.
const maxIdempotencyKeyLength = 255

// Store, handler'in ihtiyac duydugu davranislari tanimlar.
// Arayuz tuketen tarafta durur: handler repository'nin tamamini degil,
// yalnizca kullandigi iki metodu bilir.
type Store interface {
	CreateTenantProvision(ctx context.Context, req oprepo.ProvisionRequest) (*oprepo.ProvisionResult, error)
	GetOperation(ctx context.Context, id string) (*domain.Operation, error)
}

// Handler, operasyon ve tenant kurulumu uclarini saglar.
type Handler struct {
	store Store
	log   *logger.Logger
}

// New, handler olusturur.
func New(store Store, log *logger.Logger) *Handler {
	return &Handler{store: store, log: log}
}

// CreateTenantRequest, kurulum istegidir.
type CreateTenantRequest struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Email string `json:"email"`
	Plan  string `json:"plan,omitempty"`
}

// OperationResponse, operasyon kaynaginin temsilidir.
type OperationResponse struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	ErrorCode   string `json:"error_code,omitempty"`
	ErrorDetail string `json:"error_detail,omitempty"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// CreateTenant, kurulum isini kabul eder.
//
// Yanit 202 Accepted'dir ve Location basligi operasyon kaynagini gosterir.
// 201 Created donmek yanlis olurdu: tenant henuz kullanilabilir degil.
//
// Tekrarlanan istek de 202 doner ve ayni operasyonu gosterir. Istemcinin
// yeniden denemesi ikinci bir kurulum baslatmaz.
func (h *Handler) CreateTenant(c *fiber.Ctx) error {
	var req CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return problem(c, http.StatusBadRequest, "invalid_body", "İstek gövdesi okunamadı")
	}

	if msg := validateCreate(req); msg != "" {
		return problem(c, http.StatusBadRequest, "validation_failed", msg)
	}

	key := strings.TrimSpace(c.Get("Idempotency-Key"))
	if len(key) > maxIdempotencyKeyLength {
		return problem(c, http.StatusBadRequest, "idempotency_key_too_long",
			fmt.Sprintf("Idempotency-Key en fazla %d karakter olabilir", maxIdempotencyKeyLength))
	}

	result, err := h.store.CreateTenantProvision(c.UserContext(), oprepo.ProvisionRequest{
		Name:      req.Name,
		Slug:      req.Slug,
		Email:     req.Email,
		Plan:      req.Plan,
		CreatedBy: subjectID(c),
		// Kapsam, anahtari isteyen ozneye baglanir. Bir musterinin anahtari
		// digerinin istegini eslestirmemelidir.
		Scope:          "subject:" + subjectID(c),
		IdempotencyKey: key,
		RequestBody:    c.Body(),
	})
	if err != nil {
		return h.mapCreateError(c, err)
	}

	location := "/api/v1/operations/" + result.Operation.ID
	c.Set("Location", location)

	// Tekrarlanan istek acikca isaretlenir. Istemci, isteginin yeni bir islem
	// baslatmadigini gorebilmelidir.
	if result.Replayed {
		c.Set("Idempotent-Replay", "true")
	}

	return c.Status(http.StatusAccepted).JSON(fiber.Map{
		"operation": toResponse(result.Operation),
		"tenant_id": result.TenantID,
	})
}

// GetOperation, operasyon durumunu dondurur.
func (h *Handler) GetOperation(c *fiber.Ctx) error {
	op, err := h.store.GetOperation(c.UserContext(), c.Params("id"))

	switch {
	case errors.Is(err, domain.ErrNotFound):
		return problem(c, http.StatusNotFound, "operation_not_found", "Operasyon bulunamadı")
	case err != nil:
		return h.internal(c, err)
	}

	return c.JSON(toResponse(op))
}

// mapCreateError, alan hatalarini HTTP durumlarina cevirir.
func (h *Handler) mapCreateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrIdempotencyConflict):
		// 409: anahtar aynı ama gövde farklı. Sessizce eski sonucu döndürmek
		// istemciye göndermediği isteğin işlendiğini düşündürürdü.
		return problem(c, http.StatusConflict, "idempotency_key_reused",
			"Bu Idempotency-Key farklı bir istek gövdesiyle kullanılmış")

	case errors.Is(err, oprepo.ErrSlugTaken):
		return problem(c, http.StatusConflict, "slug_taken", "Bu slug kullanımda")

	default:
		return h.internal(c, err)
	}
}

// internal, beklenmeyen hatalari loglar ve genel bir yanit doner.
// Uygulama detayi istemciye sizmaz.
func (h *Handler) internal(c *fiber.Ctx, err error) error {
	if h.log != nil {
		h.log.WithFields(logger.Fields{
			"path":  c.Path(),
			"error": err.Error(),
		}).Error("İstek işlenemedi")
	}

	return problem(c, http.StatusInternalServerError, "internal_error", "İstek işlenemedi")
}

// validateCreate, zorunlu alanlari dogrular.
func validateCreate(req CreateTenantRequest) string {
	switch {
	case strings.TrimSpace(req.Name) == "":
		return "name zorunlu"
	case strings.TrimSpace(req.Slug) == "":
		return "slug zorunlu"
	case strings.TrimSpace(req.Email) == "":
		return "email zorunlu"
	}

	return ""
}

// subjectID, istegi yapan oznenin kimligini dondurur.
func subjectID(c *fiber.Ctx) string {
	id, _ := c.Locals("user_id").(string)
	return id
}

// toResponse, alan nesnesini API temsiline cevirir.
func toResponse(op *domain.Operation) OperationResponse {
	resp := OperationResponse{
		ID:          op.ID,
		TenantID:    op.TenantID,
		Kind:        string(op.Kind),
		Status:      string(op.Status),
		ErrorCode:   op.ErrorCode,
		ErrorDetail: op.ErrorMessage,
		CreatedAt:   op.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}

	if op.CompletedAt != nil {
		resp.CompletedAt = op.CompletedAt.UTC().Format("2006-01-02T15:04:05Z")
	}

	return resp
}

// problem, tutarli bir hata govdesi doner.
// Makine tarafindan okunabilir kod, istemcinin mesaj icerigine gore
// dallanmasini engeller.
func problem(c *fiber.Ctx, status int, code, detail string) error {
	return c.Status(status).JSON(fiber.Map{
		"type":   "https://keystone.dev/problems/" + code,
		"code":   code,
		"status": status,
		"detail": detail,
	})
}
