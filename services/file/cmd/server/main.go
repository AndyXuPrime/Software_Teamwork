package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/config"
	filehttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/platform/storage"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "service", "file", "error", err)
		os.Exit(1)
	}

	repo := repository.NewMemoryRepository()
	objectStore := storage.NewMemoryStore()
	documentService := service.New(repo, objectStore)
	handler := filehttp.NewServer(documentService, filehttp.Config{
		MaxUploadBytes: cfg.MaxUploadBytes,
		Logger:         logger,
	})

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("file service starting", "service", "file", "addr", cfg.HTTPAddr, "storage_backend", cfg.StorageBackend)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("file service stopped unexpectedly", "service", "file", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	logger.Info("file service shutdown started", "service", "file")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("file service shutdown failed", "service", "file", "error", err)
		os.Exit(1)
	}
	logger.Info("file service shutdown complete", "service", "file")
}
