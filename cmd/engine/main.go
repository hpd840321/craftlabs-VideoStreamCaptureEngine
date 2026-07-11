package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/craftlabs/video-stream-capture-engine/internal/api"
	"github.com/craftlabs/video-stream-capture-engine/internal/config"
	"github.com/craftlabs/video-stream-capture-engine/internal/manager"
	"github.com/craftlabs/video-stream-capture-engine/internal/store"
)

//go:embed dist
var webAssets embed.FS

var (
	version = "dev"
	commit  = "unknown"
)

func init() {
	slog.Info("engine build info", "version", version, "commit", commit)
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	slog.Info("VideoStreamCaptureEngine starting", "config", *configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	applyEnvOverrides(cfg)

	mgr, err := manager.NewStreamManager(cfg)
	if err != nil {
		slog.Error("failed to create stream manager", "error", err)
		os.Exit(1)
	}

	// Initialize PostgreSQL connection
	db, err := store.NewDB(context.Background(), cfg.Database.DSN())
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(context.Background()); err != nil {
		slog.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	// Create API server
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}
	apiServer := api.NewServer(cfg, mgr, db, jwtSecret)

	// Initialize admin user
	if err := apiServer.InitAdminUser(); err != nil {
		slog.Error("failed to init admin user", "error", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Register API routes
	apiServer.RegisterRoutes(mux)

	// Serve embedded frontend SPA
	distFS, err := fs.Sub(webAssets, "dist")
	if err != nil {
		slog.Warn("frontend not embedded, serving API only", "error", err)
	} else {
		fileServer := http.FileServer(http.FS(distFS))
		mux.Handle("/", fileServer)
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		slog.Info("HTTP server listening", "addr", ":8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := mgr.Start(ctx); err != nil {
			slog.Error("stream manager error", "error", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	slog.Info("received signal, shutting down", "signal", sig)

	cancel()
	mgr.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)

	slog.Info("shutdown complete")
}

func applyEnvOverrides(cfg *config.Config) {
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Database.Port)
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.DBName = v
	}
}
