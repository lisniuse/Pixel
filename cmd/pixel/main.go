package main

import (
	"log"

	"github.com/getlantern/systray"
	"github.com/lisniuse/pixel/internal/config"
	"github.com/lisniuse/pixel/internal/tray"
)

func main() {
	// Load (or create) ~/.desktop-pixel/config.json.
	cfg, err := config.Load()
	if err != nil {
		log.Printf("config: %v", err)
		// Fall back to defaults so the app still starts.
		cfg = &config.Config{IntervalSeconds: 10}
	}

	// Route all log output to ~/.desktop-pixel/logs/pixel-YYYY-MM-DD.log.
	logFile, err := config.SetupLogging()
	if err != nil {
		// Not fatal — keep logging to stderr.
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Printf("could not set up file logging: %v", err)
	} else {
		defer logFile.Close()
	}

	log.Println("Pixel starting")

	tray.Init(cfg)
	systray.Run(tray.OnReady, tray.OnExit)
}
