package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hoanghai1803/apricot/internal/ai"
	"github.com/hoanghai1803/apricot/internal/api"
	"github.com/hoanghai1803/apricot/internal/config"
	"github.com/hoanghai1803/apricot/internal/feeds"
	"github.com/hoanghai1803/apricot/internal/storage"
)

func main() {
	configPath := flag.String("config", "config.toml", "path to config file")
	dataDir := flag.String("data-dir", "./data", "path to data directory")
	flag.Parse()

	// Load configuration (auto-creates default if missing).
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Ensure data directory exists.
	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		slog.Error("failed to create data directory", "error", err)
		os.Exit(1)
	}

	// Open database with WAL mode and pragmas.
	db, err := storage.OpenDatabase(filepath.Join(*dataDir, "app.db"))
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run schema migrations.
	if err := storage.RunMigrations(db); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Create store and seed default blog sources.
	store := storage.NewStore(db)
	if err := store.SeedDefaults(context.Background()); err != nil {
		slog.Error("failed to seed defaults", "error", err)
		os.Exit(1)
	}

	// Create AI provider (nil if no API key -- handlers check for this).
	var aiProvider ai.AIProvider
	if cfg.AI.APIKey != "" {
		aiProvider, err = ai.NewProvider(ai.ProviderConfig{
			Provider: cfg.AI.Provider,
			APIKey:   cfg.AI.APIKey,
			Model:    cfg.AI.Model,
		})
		if err != nil {
			slog.Error("failed to create AI provider", "error", err)
			os.Exit(1)
		}
		slog.Info("AI provider configured", "provider", cfg.AI.Provider, "model", cfg.AI.Model)
	} else {
		slog.Warn("no AI provider API key configured, AI features will be disabled")
	}

	// Create feed fetcher.
	fetcher := feeds.NewFetcher()

	// Build router with all API routes and static file serving.
	router := api.NewRouter(store, aiProvider, fetcher, cfg)

	// Determine server address (localhost only for security).
	addr := fmt.Sprintf("localhost:%d", cfg.Server.Port)

	// Auto-open browser after a short delay to let the server start.
	if cfg.Server.AutoOpenBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser("http://" + addr)
		}()
	}

	// Start HTTP server.
	slog.Info("starting server", "addr", "http://"+addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// openBrowser opens the given URL in the user's default browser.
// It is a fire-and-forget operation; errors are silently ignored.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}
