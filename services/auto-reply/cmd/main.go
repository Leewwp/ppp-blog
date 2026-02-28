package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/auto-reply/internal/ai"
	"github.com/ppp-blog/auto-reply/internal/engine"
	"github.com/ppp-blog/auto-reply/internal/handler"
	"github.com/ppp-blog/auto-reply/internal/middleware"
	"github.com/ppp-blog/auto-reply/internal/store"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := getEnv("PORT", "8092")
	rulesFile := getEnv("RULES_FILE", "config/rules.json")

	aiClient := ai.NewClient(ai.ClientConfig{
		Enabled:       getEnvBool("AUTO_REPLY_AI_ENABLED", true),
		APIKey:        getEnv("MINIMAX_API_KEY", ""),
		APIURL:        getEnv("MINIMAX_API_URL", "https://api.minimaxi.chat/v1/text/chatcompletion_v2"),
		Model:         getEnv("MINIMAX_MODEL", "MiniMax-Text-01"),
		Timeout:       time.Duration(getEnvInt("AUTO_REPLY_AI_TIMEOUT_SECONDS", 10)) * time.Second,
		MaxReplyChars: getEnvInt("AUTO_REPLY_MAX_REPLY_CHARS", 120),
	}, logger)
	limiter := ai.NewQuotaLimiter(ai.QuotaLimiterConfig{
		DailyGlobalLimit: getEnvInt("AUTO_REPLY_DAILY_CALL_LIMIT", 300),
		DailyPerAuthor:   getEnvInt("AUTO_REPLY_DAILY_AUTHOR_LIMIT", 20),
		AuthorCooldown:   time.Duration(getEnvInt("AUTO_REPLY_AUTHOR_COOLDOWN_SECONDS", 60)) * time.Second,
		MaxCommentChars:  getEnvInt("AUTO_REPLY_MAX_COMMENT_CHARS", 180),
	}, logger)

	ruleStore, err := store.NewRuleStore(rulesFile, logger)
	if err != nil {
		logger.Error("failed to initialize rule store", "error", err)
		os.Exit(1)
	}

	ruleEngine, err := engine.NewRuleEngine()
	if err != nil {
		logger.Error("failed to initialize rule engine", "error", err)
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

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(logger))
	router.Use(metrics.Middleware())
	router.GET("/metrics", metrics.MetricsHandler())

	replyHandler := handler.NewReplyHandler(
		ruleStore,
		ruleEngine,
		aiClient,
		limiter,
		logger,
		metrics.RecordReplyDecision,
	)
	ruleHandler := handler.NewRuleHandler(ruleStore, logger)
	healthHandler := handler.NewHealthHandler(ruleStore)

	api := router.Group("/api/v1")
	{
		api.POST("/reply", replyHandler.Reply)
		api.GET("/rules", ruleHandler.ListRules)
		api.POST("/rules", ruleHandler.CreateRule)
		api.PUT("/rules/:id", ruleHandler.UpdateRule)
		api.DELETE("/rules/:id", ruleHandler.DeleteRule)
		api.GET("/health", healthHandler.Health)
	}

	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("auto-reply service started",
			"addr", addr,
			"rules_file", rulesFile,
			"rule_count", ruleStore.Count(),
			"ai_enabled", aiClient.Enabled(),
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
		)
	}
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
