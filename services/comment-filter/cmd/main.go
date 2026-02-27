package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/comment-filter/internal/handler"
	"github.com/ppp-blog/comment-filter/internal/middleware"
	"github.com/ppp-blog/comment-filter/internal/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := getEnv("PORT", "8091")
	wordsFile := getEnv("WORDS_FILE", "data/sensitive_words.txt")

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

	filterService := service.NewFilterService(store, logger, metrics.IncFilterHit)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("http request",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	})
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
		logger.Info("comment-filter service started", "addr", addr, "words_file", wordsFile, "word_count", store.Count())
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
