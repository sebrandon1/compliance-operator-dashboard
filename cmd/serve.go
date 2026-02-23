package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/api"
	"github.com/sebrandon1/compliance-operator-dashboard/internal/compliance"
	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
	"github.com/sebrandon1/compliance-operator-dashboard/internal/ws"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the dashboard web server",
	Long:  `Starts the HTTP server with embedded React frontend and REST API for managing the Compliance Operator.`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	// Initialize structured logging
	var handler slog.Handler
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(handler))

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient(cfg.KubeConfig)
	if err != nil {
		slog.Warn("could not connect to Kubernetes cluster", "error", err)
		slog.Info("dashboard will start but cluster features will be unavailable")
	}

	// Initialize compliance service
	complianceSvc := compliance.NewService(k8sClient, cfg.Namespace, cfg.ComplianceOpRef)

	// Initialize WebSocket hub
	hub := ws.NewHub()
	go hub.Run(ctx)

	// Start Kubernetes watchers if connected
	if k8sClient != nil {
		watcher := ws.NewWatcher(k8sClient, hub, cfg.Namespace)
		go watcher.Start(ctx)
	}

	// Create and start HTTP server
	srv := api.NewServer(cfg, complianceSvc, hub)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      srv.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutting down server")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
		cancel()
	}()

	slog.Info("starting compliance operator dashboard", "port", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
