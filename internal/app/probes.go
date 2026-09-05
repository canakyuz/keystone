package app

import (
	"context"
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"github.com/canakyuz/keystone/internal/config"
)

// probeTimeout, bagimlilik yoklamalarinin ust sinirini belirler.
//
// Kisa tutulur: yoklama, load balancer'in yoklama araligindan uzun surerse
// istekler birikir ve saglik kontrolunun kendisi bir yuk kaynagina donusur.
const probeTimeout = 2 * time.Second

// registerProbes, canlilik ve hazir olma uclarini kaydeder.
//
// Ikisi ayri uclardir cunku farkli sorulari yanitlarlar ve orkestratorde
// farkli sonuclari vardir:
//
//   - /health (liveness): surec ayakta mi? Basarisiz olursa container
//     yeniden baslatilir. Bagimliliklara BAKMAZ. Veritabani gecici olarak
//     dustugunde saglikli surecleri yeniden baslatmak, kurtarma sirasinda
//     baglanti firtinasi yaratir ve durumu kotuyestirir.
//
//   - /ready (readiness): bu surec simdi istek karsilayabilir mi?
//     Basarisiz olursa load balancer trafigi keser ama surec yasamaya
//     devam eder. Bagimliliklara BAKAR.
//
// Onceki uygulamada yalnizca /health vardi ve sabit bir JSON donduruyordu.
// Veritabani erisilemez oldugunda bile "ok" yanitladigi icin load balancer
// istek gondermeye devam ediyordu.
func registerProbes(app *fiber.App, cfg *config.Config, db *sql.DB, rdb *redis.Client) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":      "ok",
			"environment": cfg.Server.Environment,
		})
	})

	app.Get("/ready", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), probeTimeout)
		defer cancel()

		checks := map[string]string{
			"database": probeDatabase(ctx, db),
			"redis":    probeRedis(ctx, rdb),
		}

		if ready(checks) {
			return c.JSON(fiber.Map{"status": "ready", "checks": checks})
		}

		return c.Status(fiber.StatusServiceUnavailable).
			JSON(fiber.Map{"status": "not_ready", "checks": checks})
	})
}

// probeDatabase, veritabani baglantisini yoklar.
func probeDatabase(ctx context.Context, db *sql.DB) string {
	if db == nil {
		return "yapılandırılmamış"
	}
	if err := db.PingContext(ctx); err != nil {
		return "erişilemiyor"
	}

	return "ok"
}

// probeRedis, Redis baglantisini yoklar.
func probeRedis(ctx context.Context, rdb *redis.Client) string {
	if rdb == nil {
		return "yapılandırılmamış"
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		return "erişilemiyor"
	}

	return "ok"
}

// ready, servisin trafik alabilecek durumda olup olmadigini soyler.
//
// Veritabani zorunludur. Redis degildir: onbellek ve limitleyici Redis
// olmadan da surec ici yollarina duserek calisir, dolayisiyla Redis'in
// dusmesi bu instance'i havuzdan cikarmayi gerektirmez.
func ready(checks map[string]string) bool {
	return checks["database"] == "ok"
}
