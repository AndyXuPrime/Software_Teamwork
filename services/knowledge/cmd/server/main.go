package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/config"
	knowledgehttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/embedding"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/fileclient"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/parser"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/queue"
	rerankplatform "github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/rerank"
	vectorplatform "github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/vector"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "service", "knowledge", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := connectPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres connection failed", "service", "knowledge", "dependency", "postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	fileClient, err := fileclient.New(cfg.FileServiceURL, cfg.ServiceToken, nil)
	if err != nil {
		logger.Error("file client configuration failed", "service", "knowledge", "dependency", "file", "error", err)
		os.Exit(1)
	}
	documentParser, err := buildParser(cfg)
	if err != nil {
		logger.Error("parser configuration failed", "service", "knowledge", "error", err)
		os.Exit(1)
	}
	embedder, err := buildEmbedder(cfg)
	if err != nil {
		logger.Error("embedding configuration failed", "service", "knowledge", "error", err)
		os.Exit(1)
	}
	vectorIndex, err := buildVectorIndex(cfg)
	if err != nil {
		logger.Error("vector index configuration failed", "service", "knowledge", "dependency", "qdrant", "error", err)
		os.Exit(1)
	}

	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr}
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()
	asynqInspector := asynq.NewInspector(redisOpt)
	defer asynqInspector.Close()
	ingestionQueue := queue.NewAsynqQueueWithInspector(asynqClient, asynqInspector)

	repo := repository.NewPostgresRepository(pool)
	reranker, err := newReranker(cfg)
	if err != nil {
		logger.Error("reranker configuration failed", "service", "knowledge", "dependency", "ai-gateway", "error", err)
		os.Exit(1)
	}
	knowledgeService := service.NewWithDependencies(
		repo,
		fileClient,
		ingestionQueue,
		nil,
		nil,
		service.WithProcessingPipeline(fileClient, documentParser, service.NewFixedChunker()),
		service.WithVectorIndex(embedder, vectorIndex, cfg.QdrantCollection),
	)
	if reranker != nil {
		service.WithReranker(reranker)(knowledgeService)
	}
	handler := knowledgehttp.NewServer(knowledgeService, knowledgehttp.Config{
		ServiceVersion: cfg.ServiceVersion,
		Environment:    cfg.Environment,
		Logger:         logger,
		MaxUploadBytes: cfg.MaxUploadBytes,
	})

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler,
	}

	asynqServer := asynq.NewServer(redisOpt, asynq.Config{Concurrency: 1})
	asynqMux := asynq.NewServeMux()
	ingestionHandler := worker.NewIngestionHandler(knowledgeService, worker.WithLogger(logger))
	asynqMux.HandleFunc(queue.DocumentIngestionTaskType, func(ctx context.Context, task *asynq.Task) error {
		return ingestionHandler.HandleIngestionPayload(ctx, task.Payload())
	})
	deleteCleanupHandler := worker.NewDeleteCleanupHandler(knowledgeService, logger)
	asynqMux.HandleFunc(queue.DocumentDeleteCleanupTaskType, func(ctx context.Context, task *asynq.Task) error {
		return deleteCleanupHandler.HandleDeleteCleanupPayload(ctx, task.Payload())
	})

	go func() {
		logger.Info("knowledge service starting", "service", "knowledge", "addr", cfg.HTTPAddr, "environment", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("knowledge service stopped unexpectedly", "service", "knowledge", "error", err)
			stop()
		}
	}()
	go func() {
		logger.Info("knowledge workers starting", "service", "knowledge", "queues", []string{queue.DocumentIngestionTaskType, queue.DocumentDeleteCleanupTaskType})
		if err := asynqServer.Run(asynqMux); err != nil {
			logger.Error("knowledge worker stopped unexpectedly", "service", "knowledge", "error", err)
			stop()
		}
	}()
	go runDeleteCleanupReconciler(ctx, logger, knowledgeService, time.Minute, 50)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	logger.Info("knowledge service shutdown started", "service", "knowledge")
	asynqServer.Shutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("knowledge service shutdown failed", "service", "knowledge", "error", err)
		os.Exit(1)
	}
	logger.Info("knowledge service shutdown complete", "service", "knowledge")
}

func runDeleteCleanupReconciler(ctx context.Context, logger *slog.Logger, knowledge *service.Service, interval time.Duration, limit int) {
	if knowledge == nil || interval <= 0 || limit <= 0 {
		return
	}
	reconcile := func() {
		reconcileCtx, cancel := context.WithTimeout(ctx, minDuration(interval, 30*time.Second))
		defer cancel()
		result, err := knowledge.RequeueDeleteCleanupTasks(reconcileCtx, service.RequestContext{
			RequestID:     "delete_cleanup_reconciler",
			CallerService: "knowledge",
		}, limit)
		if err != nil {
			errorCode := "unknown"
			if appErr, ok := service.Classify(err); ok {
				errorCode = string(appErr.Code)
			}
			dependency := strings.TrimSpace(result.FailedDependency)
			if dependency == "" {
				dependency = "unknown"
			}
			logger.WarnContext(reconcileCtx, "knowledge delete cleanup requeue failed",
				"service", "knowledge",
				"operation", "knowledge_delete_cleanup_reconciler",
				"dependency", dependency,
				"status", "failed",
				"scanned", result.Scanned,
				"enqueued", result.Enqueued,
				"failed", result.Failed,
				"error_code", errorCode,
			)
			return
		}
		if result.Enqueued > 0 {
			logger.InfoContext(reconcileCtx, "knowledge delete cleanup tasks requeued",
				"service", "knowledge",
				"operation", "knowledge_delete_cleanup_reconciler",
				"status", "success",
				"scanned", result.Scanned,
				"enqueued", result.Enqueued,
			)
		}
	}

	reconcile()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcile()
		}
	}
}

func minDuration(a time.Duration, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func connectPostgres(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func buildParser(cfg config.Config) (service.Parser, error) {
	return parser.NewServiceClient(parser.ServiceClientConfig{
		BaseURL:      cfg.ParserServiceBaseURL,
		ServiceToken: cfg.ParserServiceToken,
		Timeout:      cfg.ParserServiceTimeout,
	})
}

func buildEmbedder(cfg config.Config) (service.Embedder, error) {
	if strings.EqualFold(strings.TrimSpace(cfg.EmbeddingProvider), "ai_gateway") {
		return embedding.NewAIGatewayClient(embedding.AIGatewayConfig{
			BaseURL:      cfg.AIGatewayBaseURL,
			Model:        cfg.EmbeddingModel,
			ProfileID:    cfg.AIGatewayProfileID,
			Dimensions:   cfg.EmbeddingDimension,
			ServiceToken: cfg.AIGatewayToken,
		})
	}
	return embedding.NewLocalHasher(cfg.EmbeddingProvider, cfg.EmbeddingModel, cfg.EmbeddingDimension), nil
}

func buildVectorIndex(cfg config.Config) (service.VectorIndex, error) {
	if strings.TrimSpace(cfg.QdrantURL) == "" {
		return vectorplatform.NewMemoryIndex(), nil
	}
	return vectorplatform.NewQdrantClient(vectorplatform.QdrantConfig{
		BaseURL:    cfg.QdrantURL,
		APIKey:     cfg.QdrantAPIKey,
		Collection: cfg.QdrantCollection,
		Dimension:  cfg.EmbeddingDimension,
	})
}

func newReranker(cfg config.Config) (service.Reranker, error) {
	if strings.TrimSpace(cfg.RerankModel) == "" {
		return nil, nil
	}
	return rerankplatform.NewAIGatewayReranker(rerankplatform.AIGatewayConfig{
		BaseURL:      cfg.AIGatewayBaseURL,
		ServiceToken: cfg.AIGatewayToken,
		Model:        cfg.RerankModel,
		ProfileID:    cfg.RerankProfileID,
	})
}
