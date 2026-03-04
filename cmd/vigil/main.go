package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vigil/internal/api"
	"vigil/internal/autodiscovery"
	"vigil/internal/config"
	"vigil/internal/db"
	"vigil/internal/evaluator"
	"vigil/internal/grafana"
)

func main() {
	cfg := config.Load()
	log.Printf("starting vigil — listen=%s eval_interval=%s", cfg.ListenAddr, cfg.EvalInterval)

	// Database
	database := db.Open(cfg.DBPath)

	// Grafana client (optional)
	grafanaClient := grafana.NewClient(cfg.GrafanaURL, cfg.GrafanaAPIToken)
	_ = grafanaClient // available for future use in evaluator state change hooks

	// Shared clients
	promClient := evaluator.NewPromClient(cfg.PrometheusURL, cfg.PrometheusUser, cfg.PrometheusPassword)
	lokiClient := evaluator.NewLokiClient(cfg.LokiURL, cfg.LokiUser, cfg.LokiPassword)

	// Evaluator
	eval := evaluator.New(database, cfg, promClient, lokiClient)

	// Auto-discovery (runs every hour)
	disc := autodiscovery.New(database, lokiClient, 1*time.Hour)

	// HTTP server
	server := api.NewServer(database, promClient, lokiClient)
	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: server.Handler(),
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Start evaluator
	go eval.Run(ctx)

	// Start auto-discovery
	go disc.Run(ctx)

	// Start HTTP server
	go func() {
		log.Printf("http server listening on %s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("received signal %v, shutting down", sig)

	cancel() // stop evaluator + auto-discovery

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}

	log.Println("vigil stopped")
}
