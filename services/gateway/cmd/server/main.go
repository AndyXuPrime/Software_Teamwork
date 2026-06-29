package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/config"
	gatewayhttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/http"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "service", "gateway", "error", err)
		os.Exit(1)
	}

	handler := gatewayhttp.NewServer(gatewayhttp.Config{
		Logger:               logger,
		ServiceVersion:       cfg.ServiceVersion,
		Environment:          cfg.Environment,
		RequestTimeout:       cfg.RequestTimeout,
		MaxBodyBytes:         cfg.MaxBodyBytes,
		CORSAllowedOrigins:   cfg.CORSAllowedOrigins,
		CORSAllowedMethods:   cfg.CORSAllowedMethods,
		CORSAllowedHeaders:   cfg.CORSAllowedHeaders,
		CORSAllowCredentials: cfg.CORSAllowCredentials,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("gateway service starting",
			"service", "gateway",
			"addr", cfg.HTTPAddr,
			"environment", cfg.Environment,
			"version", cfg.ServiceVersion,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("gateway service stopped unexpectedly", "service", "gateway", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	logger.Info("gateway service shutdown started", "service", "gateway")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("gateway service shutdown failed", "service", "gateway", "error", err)
		os.Exit(1)
	}
	logger.Info("gateway service shutdown complete", "service", "gateway")
}
