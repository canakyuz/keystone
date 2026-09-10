package app

import (
	"context"
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"github.com/canakyuz/keystone/internal/config"
)

// probeTimeout caps how long a dependency probe may take.
//
// It is kept short: if a probe outlasts the load balancer's probe interval, requests
// pile up and the health check itself becomes a source of load.
const probeTimeout = 2 * time.Second

// registerProbes registers the liveness and readiness endpoints.
//
// They are separate endpoints because they answer different questions and have
// different consequences in the orchestrator:
//
//   - /health (liveness): is the process alive? On failure the container is
//     restarted. It does NOT touch dependencies. Restarting healthy processes because
//     the database blipped creates a connection storm during recovery and makes the
//     situation worse.
//
//   - /ready (readiness): can this process serve requests right now? On failure the
//     load balancer drains traffic, but the process keeps living. It DOES check
//     dependencies.
//
// The previous implementation had only /health and returned a constant JSON body.
// Because it answered "ok" even when the database was unreachable, the load balancer
// kept sending it requests.
// Both are registered before the global middleware, which in Fiber means none of it runs
// for them. That is deliberate for a probe and only for a probe: an orchestrator polls
// these every few seconds forever, so counting them would swamp the request rate, tracing
// them would make liveness checks the bulk of the trace volume, and rate limiting them
// would pull a healthy instance out of the pool for answering too often.
//
// Nothing else belongs on that side of the middleware. The provisioning endpoints were
// registered there too, and the effect was that the most important endpoint in the system
// had no metrics, no traces and no rate limit.
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

// probeDatabase probes the database connection.
func probeDatabase(ctx context.Context, db *sql.DB) string {
	if db == nil {
		return "not configured"
	}
	if err := db.PingContext(ctx); err != nil {
		return "unreachable"
	}

	return "ok"
}

// probeRedis probes the Redis connection.
func probeRedis(ctx context.Context, rdb *redis.Client) string {
	if rdb == nil {
		return "not configured"
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		return "unreachable"
	}

	return "ok"
}

// ready says whether the service is in a state to take traffic.
//
// The database is required. Redis is not: the cache and the limiter fall back to
// their in-process paths without it, so Redis being down is no reason to pull this
// instance out of the pool.
func ready(checks map[string]string) bool {
	return checks["database"] == "ok"
}
