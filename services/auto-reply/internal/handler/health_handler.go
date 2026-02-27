package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/auto-reply/internal/store"
)

type HealthHandler struct {
	ruleStore *store.RuleStore
}

func NewHealthHandler(ruleStore *store.RuleStore) *HealthHandler {
	return &HealthHandler{ruleStore: ruleStore}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"rule_count": h.ruleStore.Count(),
	})
}
