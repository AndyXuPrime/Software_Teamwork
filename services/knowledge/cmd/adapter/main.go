package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/adapter"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/adapterconfig"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := adapterconfig.Load()
	if err != nil {
		logger.Error("configuration failed", "service", "knowledge-adapter", "error", err)
		os.Exit(1)
	}

	server := adapter.NewServer(cfg, logger)
	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: server.Handler(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("knowledge adapter listening", "addr", cfg.HTTPAddr, "vendor_runtime_url", cfg.VendorRuntimeURL)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("adapter server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("adapter shutdown failed", "error", err)
		os.Exit(1)
	}
}
