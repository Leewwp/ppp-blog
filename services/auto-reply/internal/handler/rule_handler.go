package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/auto-reply/internal/engine"
	"github.com/ppp-blog/auto-reply/internal/store"
)

type RuleHandler struct {
	ruleStore *store.RuleStore
	logger    *slog.Logger
}

type createRuleRequest struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Keywords  []string         `json:"keywords"`
	TimeRange engine.TimeRange `json:"time_range"`
	Templates []string         `json:"templates"`
	Priority  int              `json:"priority"`
	Enabled   *bool            `json:"enabled"`
}

type updateRuleRequest struct {
	Name      *string           `json:"name"`
	Keywords  *[]string         `json:"keywords"`
	TimeRange *engine.TimeRange `json:"time_range"`
	Templates *[]string         `json:"templates"`
	Priority  *int              `json:"priority"`
	Enabled   *bool             `json:"enabled"`
}

func NewRuleHandler(ruleStore *store.RuleStore, logger *slog.Logger) *RuleHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &RuleHandler{
		ruleStore: ruleStore,
		logger:    logger,
	}
}

func (h *RuleHandler) ListRules(c *gin.Context) {
	rules := h.ruleStore.ListRules()
	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
		"count": len(rules),
	})
}

func (h *RuleHandler) CreateRule(c *gin.Context) {
	var req createRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid create rule request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	created, err := h.ruleStore.CreateRule(engine.Rule{
		ID:        strings.TrimSpace(req.ID),
		Name:      strings.TrimSpace(req.Name),
		Keywords:  req.Keywords,
		TimeRange: req.TimeRange,
		Templates: req.Templates,
		Priority:  req.Priority,
		Enabled:   enabled,
	})
	if err != nil {
		h.logger.Error("create rule failed", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *RuleHandler) UpdateRule(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule id is required"})
		return
	}

	existing, ok := h.ruleStore.GetRule(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	var req updateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid update rule request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	if req.Name != nil {
		existing.Name = strings.TrimSpace(*req.Name)
	}
	if req.Keywords != nil {
		existing.Keywords = *req.Keywords
	}
	if req.TimeRange != nil {
		existing.TimeRange = *req.TimeRange
	}
	if req.Templates != nil {
		existing.Templates = *req.Templates
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	updated, err := h.ruleStore.UpdateRule(id, existing)
	if err != nil {
		h.logger.Error("update rule failed", "id", id, "error", err)
		if errors.Is(err, store.ErrRuleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *RuleHandler) DeleteRule(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule id is required"})
		return
	}

	if err := h.ruleStore.DeleteRule(id); err != nil {
		if errors.Is(err, store.ErrRuleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
			return
		}
		h.logger.Error("delete rule failed", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "rule deleted",
		"id":      id,
	})
}
