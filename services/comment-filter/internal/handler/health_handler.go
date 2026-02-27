package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/comment-filter/internal/service"
)

type HealthHandler struct {
	store *service.WordStore
}

func NewHealthHandler(store *service.WordStore) *HealthHandler {
	return &HealthHandler{store: store}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"word_count": h.store.Count(),
	})
}
