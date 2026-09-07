package app

import (
	"github.com/gofiber/fiber/v2"

	operationHandler "github.com/canakyuz/keystone/internal/handler/operation"
	"github.com/canakyuz/keystone/internal/middleware"
)

// registerOperationRoutes, tenant kurulumu ve operasyon sorgulama uclarini kaydeder.
//
// NEDEN setupRoutes icinde degil: o fonksiyonun parametre listesi zaten yirmi
// ustunde. Yeni bir bagimlilik eklemek listeyi daha da buyuturdu. Bu uclar
// kendi bagimliliklarini tasiyan ayri bir fonksiyonda duruyor.
//
// TENANT CONTEXT YOK
// POST /tenants, tenantContextMiddleware kullanmaz. Gerekce: tenant henuz
// olusmamistir. Middleware var olmayan bir tenant'in semasini cozmeye calisir
// ve istek 404 ile duserdi.
//
// Bu ucun yetkilendirmesi bu yuzden farkli calisir: kimlik dogrulanir, ama
// tenant uyeligi aranmaz. Yeni tenant olusturma yetkisi platform seviyesinde
// bir karardir; su an yalnizca kimlik dogrulamasi ile korunuyor ve bu bir
// eksiklik olarak INVARIANTS.md'de kayitli.
func registerOperationRoutes(app *fiber.App, jwtSecret string, h *operationHandler.Handler) {
	auth := middleware.AuthMiddleware(jwtSecret)

	v1 := app.Group("/api/v1")

	v1.Post("/tenants", auth, h.CreateTenant)
	v1.Get("/operations/:id", auth, h.GetOperation)
}
