package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/comment-filter/internal/handler"
	"github.com/ppp-blog/comment-filter/internal/middleware"
	"github.com/ppp-blog/comment-filter/internal/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const (
	traceIDHeader     = "X-Trace-Id"
	requestIDHeader   = "X-Request-Id"
	traceIDContextKey = "trace_id"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := getEnv("PORT", "8091")
	wordsFile := getEnv("WORDS_FILE", "data/sensitive_words.txt")

	reviewer := service.NewAIReviewer(service.ReviewerConfig{
		Enabled:        getEnvBool("COMMENT_REVIEW_AI_ENABLED", true),
		APIKey:         getEnv("MINIMAX_API_KEY", ""),
		APIURL:         getEnv("MINIMAX_API_URL", "https://api.minimaxi.com/anthropic/v1/messages"),
		Model:          getEnv("MINIMAX_MODEL", "MiniMax-M2.5"),
		Timeout:        time.Duration(getEnvInt("COMMENT_REVIEW_AI_TIMEOUT_SECONDS", 30)) * time.Second,
		MaxContentChar: getEnvInt("COMMENT_REVIEW_AI_MAX_CONTENT_CHARS", 500),
	}, logger)

	store, err := service.NewWordStore(wordsFile, logger)
	if err != nil {
		logger.Error("failed to initialize word store", "error", err)
		os.Exit(1)
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	metrics, err := middleware.NewMetrics(registry)
	if err != nil {
		logger.Error("failed to initialize metrics", "error", err)
		os.Exit(1)
	}

	filterService := service.NewFilterService(store, reviewer, logger, metrics.IncFilterHit)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(traceIDMiddleware())
	router.Use(requestLogger(logger))
	router.Use(metrics.Middleware())
	router.GET("/metrics", metrics.MetricsHandler())

	api := router.Group("/api/v1")
	{
		filterHandler := handler.NewFilterHandler(filterService, logger)
		wordHandler := handler.NewWordHandler(store, logger)
		healthHandler := handler.NewHealthHandler(store)

		api.POST("/filter", filterHandler.Filter)
		api.GET("/words", wordHandler.ListWords)
		api.POST("/words", wordHandler.AddWord)
		api.DELETE("/words", wordHandler.DeleteWord)
		api.GET("/health", healthHandler.Health)
	}

	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("comment-filter service started",
			"addr", addr,
			"words_file", wordsFile,
			"word_count", store.Count(),
			"ai_review_enabled", reviewer.Enabled(),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		logger.Info("http request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"trace_id", getTraceID(c),
		)
	}
}

func traceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := strings.TrimSpace(c.GetHeader(traceIDHeader))
		if traceID == "" {
			traceID = strings.TrimSpace(c.GetHeader(requestIDHeader))
		}
		if traceID == "" {
			traceID = newTraceID()
		}

		c.Set(traceIDContextKey, traceID)
		c.Request.Header.Set(traceIDHeader, traceID)
		c.Writer.Header().Set(traceIDHeader, traceID)
		c.Next()
	}
}

func getTraceID(c *gin.Context) string {
	if value, ok := c.Get(traceIDContextKey); ok {
		if traceID, castOK := value.(string); castOK && traceID != "" {
			return traceID
		}
	}
	return ""
}

func newTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}
