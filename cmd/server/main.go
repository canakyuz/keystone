// Command server runs the control plane together with the example modules.
//
// The two halves are composed here and nowhere else. internal/app builds the control
// plane and has no path to examples/verticals; this file is the only place that knows
// about both, which is what keeps the dependency running one way.
//
// A deployment that wants the control plane alone drops the verticals.Register call. The
// modules go with it, and nothing in internal/ notices.
package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/canakyuz/keystone/examples/verticals"
	"github.com/canakyuz/keystone/internal/app"
	"github.com/canakyuz/keystone/internal/config"
)

func main() {
	// The .env file is optional; in production the values come from the environment.
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	application, err := app.NewApplication(cfg)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	// Mounted after the control plane is built and before it starts listening, so the
	// example routes sit behind the same middleware chain as everything else.
	extension := application.Extension()
	verticals.Register(extension, verticals.Build(extension.DB, cfg, extension.Logger))

	if err := application.Start(); err != nil {
		log.Fatalf("application error: %v", err)
	}
}
