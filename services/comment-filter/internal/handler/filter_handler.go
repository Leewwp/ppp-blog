package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/comment-filter/internal/service"
)

type FilterHandler struct {
	filterService *service.FilterService
	logger        *slog.Logger
}

type filterRequest struct {
	Content string `json:"content"`
	Author  string `json:"author"`
}

func NewFilterHandler(filterService *service.FilterService, logger *slog.Logger) *FilterHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &FilterHandler{
		filterService: filterService,
		logger:        logger,
	}
}

func (h *FilterHandler) Filter(c *gin.Context) {
	var req filterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid filter request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	result := h.filterService.Filter(req.Content, req.Author)
	c.JSON(http.StatusOK, result)
}
