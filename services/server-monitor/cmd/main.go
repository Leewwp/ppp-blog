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
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/server-monitor/internal/alerter"
	"github.com/ppp-blog/server-monitor/internal/collector"
	"github.com/ppp-blog/server-monitor/internal/exporter"
	"github.com/ppp-blog/server-monitor/internal/handler"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := getEnv("PORT", "8093")
	webhookURL := getEnv("WEBHOOK_URL", "")
	collectIntervalSeconds := getEnvInt("COLLECT_INTERVAL_SECONDS", 30, logger)
	alertCooldownMinutes := getEnvInt("ALERT_COOLDOWN_MINUTES", 5, logger)
	rulesFile := getEnv("ALERT_RULES_FILE", "config/alert_rules.yaml")

	collectorSvc := collector.NewSystemCollector(
		time.Duration(collectIntervalSeconds)*time.Second,
		collector.DefaultHistorySize,
		logger,
	)
	engine, err := alerter.NewEngine(rulesFile, time.Duration(alertCooldownMinutes)*time.Minute, logger)
	if err != nil {
		logger.Error("failed to initialize alert engine", "error", err)
		os.Exit(1)
	}
	notifier := alerter.NewWebhookNotifier(webhookURL, logger)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	promExporter, err := exporter.NewPrometheusExporter(registry)
	if err != nil {
		logger.Error("failed to initialize prometheus exporter", "error", err)
		os.Exit(1)
	}

	appCtx, appCancel := context.WithCancel(context.Background())
	updates := collectorSvc.Start(appCtx)

	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		for {
			select {
			case <-appCtx.Done():
				return
			case snapshot, ok := <-updates:
				if !ok {
					return
				}

				promExporter.Update(snapshot)

				events := engine.Evaluate(snapshot)
				for _, event := range events {
					if !event.ShouldNotify || !notifier.Enabled() {
						continue
					}

					notifyCtx, cancelNotify := context.WithTimeout(appCtx, 15*time.Second)
					if err := notifier.Notify(notifyCtx, event); err != nil {
						logger.Warn("send webhook alert failed", "error", err, "rule", event.Alert.RuleName, "type", event.Type)
					}
					cancelNotify()
				}
			}
		}
	}()

	monitorHandler := handler.NewMonitorHandler(collectorSvc, engine, logger)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(logger))

	router.GET("/metrics", promExporter.Handler())
	api := router.Group("/api/v1")
	{
		api.GET("/metrics", monitorHandler.GetMetrics)
		api.GET("/metrics/history", monitorHandler.GetMetricsHistory)
		api.GET("/alerts", monitorHandler.GetCurrentAlerts)
		api.GET("/alerts/rules", monitorHandler.GetAlertRules)
		api.POST("/alerts/rules", monitorHandler.PostAlertRules)
		api.GET("/health", monitorHandler.Health)
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
		logger.Info("server-monitor started",
			"addr", addr,
			"collect_interval_seconds", collectIntervalSeconds,
			"alert_cooldown_minutes", alertCooldownMinutes,
			"rules_file", rulesFile,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()
	logger.Info("shutdown signal received")

	appCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	done := make(chan struct{})
	go func() {
		workers.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("background workers stopped")
	case <-time.After(5 * time.Second):
		logger.Warn("timed out waiting for background workers")
	}

	logger.Info("server-monitor stopped")
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
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int, logger *slog.Logger) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		logger.Warn("invalid integer env, use fallback", "key", key, "value", raw, "fallback", fallback)
		return fallback
	}
	return value
}
