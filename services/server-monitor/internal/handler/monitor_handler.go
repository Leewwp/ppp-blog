package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/server-monitor/internal/alerter"
	"github.com/ppp-blog/server-monitor/internal/collector"
)

type MonitorHandler struct {
	collector *collector.SystemCollector
	engine    *alerter.Engine
	logger    *slog.Logger
}

func NewMonitorHandler(collector *collector.SystemCollector, engine *alerter.Engine, logger *slog.Logger) *MonitorHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &MonitorHandler{
		collector: collector,
		engine:    engine,
		logger:    logger,
	}
}

func (h *MonitorHandler) GetMetrics(c *gin.Context) {
	var currentPtr *collector.SystemSnapshot
	if current, ok := h.collector.Current(); ok {
		currentCopy := current
		currentPtr = &currentCopy
	}

	history5m := h.collector.HistorySince(5 * time.Minute)
	c.JSON(http.StatusOK, gin.H{
		"current":    currentPtr,
		"history_5m": history5m,
		"count_5m":   len(history5m),
	})
}

func (h *MonitorHandler) GetMetricsHistory(c *gin.Context) {
	history := h.collector.History(60)
	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"count":   len(history),
	})
}

func (h *MonitorHandler) GetCurrentAlerts(c *gin.Context) {
	alerts := h.engine.CurrentFiringAlerts()
	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

func (h *MonitorHandler) GetAlertRules(c *gin.Context) {
	rules := h.engine.ListRules()
	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
		"count": len(rules),
	})
}

// PostAlertRules supports:
// 1) Single rule object -> upsert by name
// 2) Array of rule objects -> replace full rule set
func (h *MonitorHandler) PostAlertRules(c *gin.Context) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request body is required"})
		return
	}

	if strings.HasPrefix(trimmed, "[") {
		var rules []alerter.Rule
		if err := json.Unmarshal(raw, &rules); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rules array payload"})
			return
		}
		if err := h.engine.ReplaceRules(rules); err != nil {
			h.logger.Error("replace rules failed", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"rules": h.engine.ListRules(), "count": len(rules)})
		return
	}

	var rule alerter.Rule
	if err := json.Unmarshal(raw, &rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule payload"})
		return
	}
	saved, err := h.engine.UpsertRule(rule)
	if err != nil {
		h.logger.Error("upsert rule failed", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, saved)
}

func (h *MonitorHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"rule_count": len(h.engine.ListRules()),
	})
}
