package operation

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/canakyuz/keystone/internal/domain/operation"
	oprepo "github.com/canakyuz/keystone/internal/repository/operation"
)

// stubStore, HTTP katmaninin davranisini yalitmak icin kullanilir.
//
// Veritabani garantileri burada sinanmaz; onlar
// internal/repository/operation testlerinde gercek PostgreSQL'e karsi
// dogrulanir. Burada sinanan sey durum kodu, baslik ve hata eslemesi.
type stubStore struct {
	result *oprepo.ProvisionResult
	op     *domain.Operation
	err    error

	lastRequest oprepo.ProvisionRequest
}

func (s *stubStore) CreateTenantProvision(
	_ context.Context, req oprepo.ProvisionRequest,
) (*oprepo.ProvisionResult, error) {
	s.lastRequest = req
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func (s *stubStore) GetOperation(_ context.Context, _ string) (*domain.Operation, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.op, nil
}

func sampleOperation() *domain.Operation {
	return &domain.Operation{
		ID:        "0f2f7b52-0000-4000-8000-000000000001",
		TenantID:  "0f2f7b52-0000-4000-8000-000000000002",
		Kind:      domain.KindTenantProvision,
		Status:    domain.StatusPending,
		CreatedAt: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC),
	}
}

func newApp(store Store) *fiber.App {
	app := fiber.New()
	h := New(store, nil)

	app.Post("/api/v1/tenants", func(c *fiber.Ctx) error {
		c.Locals("user_id", "subject-1")
		return h.CreateTenant(c)
	})
	app.Get("/api/v1/operations/:id", h.GetOperation)

	return app
}

func post(t *testing.T, app *fiber.App, body string, headers map[string]string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := app.Test(req)
	require.NoError(t, err)

	return resp
}

func decode(t *testing.T, r io.Reader) map[string]any {
	t.Helper()

	var out map[string]any
	require.NoError(t, json.NewDecoder(r).Decode(&out))

	return out
}

const validBody = `{"name":"Acme","slug":"acme","email":"acme@example.com"}`

// TestCreateTenant_Returns202WithLocation, isin kabul edildigini ve
// operasyon adresinin verildigini dogrular.
//
// 201 Created donmek yanlis olurdu: tenant henuz kullanilabilir degil.
func TestCreateTenant_Returns202WithLocation(t *testing.T) {
	op := sampleOperation()
	store := &stubStore{result: &oprepo.ProvisionResult{Operation: op, TenantID: op.TenantID}}

	resp := post(t, newApp(store), validBody, map[string]string{"Idempotency-Key": "anahtar-1"})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Equal(t, "/api/v1/operations/"+op.ID, resp.Header.Get("Location"))
	assert.Empty(t, resp.Header.Get("Idempotent-Replay"))

	body := decode(t, resp.Body)
	operation := body["operation"].(map[string]any)
	assert.Equal(t, "pending", operation["status"])
	assert.Equal(t, op.TenantID, body["tenant_id"])
}

// TestCreateTenant_PassesIdempotencyKeyAndScope, anahtarin ve kapsamin
// depoya iletildigini dogrular.
//
// Kapsam ozneye baglanir: bir musterinin anahtari digerinin istegini
// eslestirmemelidir.
func TestCreateTenant_PassesIdempotencyKeyAndScope(t *testing.T) {
	op := sampleOperation()
	store := &stubStore{result: &oprepo.ProvisionResult{Operation: op, TenantID: op.TenantID}}

	resp := post(t, newApp(store), validBody, map[string]string{"Idempotency-Key": "  anahtar-2  "})
	defer resp.Body.Close()

	assert.Equal(t, "anahtar-2", store.lastRequest.IdempotencyKey, "anahtar kirpilmadi")
	assert.Equal(t, "subject:subject-1", store.lastRequest.Scope)
	assert.NotEmpty(t, store.lastRequest.RequestBody, "parmak izi icin govde iletilmedi")
}

// TestCreateTenant_ReplayIsMarked, tekrarlanan istegin isaretlendigini
// dogrular. Istemci isteginin yeni bir islem baslatmadigini gorebilmelidir.
func TestCreateTenant_ReplayIsMarked(t *testing.T) {
	op := sampleOperation()
	store := &stubStore{result: &oprepo.ProvisionResult{
		Operation: op, TenantID: op.TenantID, Replayed: true,
	}}

	resp := post(t, newApp(store), validBody, map[string]string{"Idempotency-Key": "anahtar-3"})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Equal(t, "true", resp.Header.Get("Idempotent-Replay"))
	assert.Equal(t, "/api/v1/operations/"+op.ID, resp.Header.Get("Location"))
}

// TestCreateTenant_IdempotencyConflictReturns409, ayni anahtarin farkli
// govdeyle kullanilmasinin 409 dondurdugunu dogrular.
func TestCreateTenant_IdempotencyConflictReturns409(t *testing.T) {
	store := &stubStore{err: domain.ErrIdempotencyConflict}

	resp := post(t, newApp(store), validBody, map[string]string{"Idempotency-Key": "anahtar-4"})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	body := decode(t, resp.Body)
	assert.Equal(t, "idempotency_key_reused", body["code"])
}

// TestCreateTenant_SlugTakenReturns409, kullanimdaki slug'in 409 dondurdugunu
// dogrular.
func TestCreateTenant_SlugTakenReturns409(t *testing.T) {
	store := &stubStore{err: oprepo.ErrSlugTaken}

	resp := post(t, newApp(store), validBody, nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	body := decode(t, resp.Body)
	assert.Equal(t, "slug_taken", body["code"])
}

// TestCreateTenant_ValidationErrors, eksik alanlarin 400 dondurdugunu dogrular.
func TestCreateTenant_ValidationErrors(t *testing.T) {
	cases := map[string]string{
		"name eksik":  `{"slug":"acme","email":"a@example.com"}`,
		"slug eksik":  `{"name":"Acme","email":"a@example.com"}`,
		"email eksik": `{"name":"Acme","slug":"acme"}`,
		"bos govde":   `{}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			store := &stubStore{result: &oprepo.ProvisionResult{Operation: sampleOperation()}}

			resp := post(t, newApp(store), body, nil)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			decoded := decode(t, resp.Body)
			assert.Equal(t, "validation_failed", decoded["code"])
		})
	}
}

// TestCreateTenant_RejectsOverlongIdempotencyKey, sinir asan anahtarin
// veritabani hatasi yerine dogrulama hatasi dondurdugunu dogrular.
func TestCreateTenant_RejectsOverlongIdempotencyKey(t *testing.T) {
	store := &stubStore{result: &oprepo.ProvisionResult{Operation: sampleOperation()}}

	long := strings.Repeat("k", maxIdempotencyKeyLength+1)
	resp := post(t, newApp(store), validBody, map[string]string{"Idempotency-Key": long})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decode(t, resp.Body)
	assert.Equal(t, "idempotency_key_too_long", body["code"])
}

// TestCreateTenant_WithoutIdempotencyKeyIsAllowed, anahtarsiz istegin kabul
// edildigini dogrular. Anahtar zorunlu degil; olmadiginda tekrar korumasi
// uygulanmaz ve bu bilincli bir tercih.
func TestCreateTenant_WithoutIdempotencyKeyIsAllowed(t *testing.T) {
	op := sampleOperation()
	store := &stubStore{result: &oprepo.ProvisionResult{Operation: op, TenantID: op.TenantID}}

	resp := post(t, newApp(store), validBody, nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Empty(t, store.lastRequest.IdempotencyKey)
}

// TestGetOperation_ReturnsStatus, operasyon durumunun sorgulanabildigini
// dogrular.
func TestGetOperation_ReturnsStatus(t *testing.T) {
	completed := time.Date(2026, 9, 7, 12, 5, 0, 0, time.UTC)
	op := sampleOperation()
	op.Status = domain.StatusSucceeded
	op.CompletedAt = &completed

	app := newApp(&stubStore{op: op})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/operations/"+op.ID, nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := decode(t, resp.Body)
	assert.Equal(t, "succeeded", body["status"])
	assert.Equal(t, "2026-09-07T12:05:00Z", body["completed_at"])
}

// TestGetOperation_NotFoundReturns404, olmayan operasyonun 404 dondurdugunu
// dogrular.
func TestGetOperation_NotFoundReturns404(t *testing.T) {
	app := newApp(&stubStore{err: domain.ErrNotFound})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/operations/yok", nil))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	body := decode(t, resp.Body)
	assert.Equal(t, "operation_not_found", body["code"])
}
