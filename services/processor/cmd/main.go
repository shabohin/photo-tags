package main

import (
	"log"
	"os"

	"github.com/shabohin/photo-tags/services/processor/internal/app"
	"github.com/shabohin/photo-tags/services/processor/internal/config"
)

func main() {
	// Load configuration
	cfg := config.New()

	// Initialize application
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Start the application
	if err := application.Start(); err != nil {
		application.Shutdown()
		log.Fatalf("Application error: %v", err)
		os.Exit(1)
	}
}
