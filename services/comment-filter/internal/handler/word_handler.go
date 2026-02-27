package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/comment-filter/internal/service"
)

type WordHandler struct {
	store  *service.WordStore
	logger *slog.Logger
}

type wordRequest struct {
	Word string `json:"word"`
}

func NewWordHandler(store *service.WordStore, logger *slog.Logger) *WordHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WordHandler{
		store:  store,
		logger: logger,
	}
}

func (h *WordHandler) ListWords(c *gin.Context) {
	words := h.store.ListWords()
	c.JSON(http.StatusOK, gin.H{
		"words": words,
		"count": len(words),
	})
}

func (h *WordHandler) AddWord(c *gin.Context) {
	word, err := parseWord(c)
	if err != nil {
		h.logger.Error("invalid add word request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.AddWord(word); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "word added", "word": strings.TrimSpace(word)})
}

func (h *WordHandler) DeleteWord(c *gin.Context) {
	word, err := parseWord(c)
	if err != nil {
		h.logger.Error("invalid delete word request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if ok := h.store.RemoveWord(word); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "word not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "word removed", "word": strings.TrimSpace(word)})
}

func parseWord(c *gin.Context) (string, error) {
	if q := strings.TrimSpace(c.Query("word")); q != "" {
		return q, nil
	}

	var req wordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return "", errors.New("word is required")
		}
		return "", errors.New("invalid request payload")
	}
	word := strings.TrimSpace(req.Word)
	if word == "" {
		return "", errors.New("word is required")
	}
	return word, nil
}
