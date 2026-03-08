package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/comment-service/internal/model"
	"github.com/ppp-blog/comment-service/internal/service"
)

// LikeHandler 提供点赞接口。
type LikeHandler struct {
	svc *service.LikeService
}

// NewLikeHandler 创建点赞处理器。
func NewLikeHandler(svc *service.LikeService) *LikeHandler {
	return &LikeHandler{svc: svc}
}

// Like 执行点赞。
func (h *LikeHandler) Like(c *gin.Context) {
	commentID := strings.TrimSpace(c.Param("comment_id"))
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment_id is required"})
		return
	}

	var req model.LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.svc.Like(c.Request.Context(), commentID, strings.TrimSpace(req.PostID), userIdentifier(c))
	if err != nil {
		handleLikeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "liked"})
}

// Unlike 取消点赞。
func (h *LikeHandler) Unlike(c *gin.Context) {
	commentID := strings.TrimSpace(c.Param("comment_id"))
	postID := strings.TrimSpace(c.Query("post_id"))
	if commentID == "" || postID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment_id and post_id are required"})
		return
	}

	err := h.svc.Unlike(c.Request.Context(), commentID, postID, userIdentifier(c))
	if err != nil {
		handleLikeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unliked"})
}

// HasLiked 查询是否已点赞。
func (h *LikeHandler) HasLiked(c *gin.Context) {
	commentID := strings.TrimSpace(c.Param("comment_id"))
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment_id is required"})
		return
	}

	liked, err := h.svc.HasLiked(c.Request.Context(), commentID, userIdentifier(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query like state failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comment_id": commentID, "liked": liked})
}

func handleLikeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAlreadyLiked):
		c.JSON(http.StatusConflict, gin.H{"error": "already liked"})
	case errors.Is(err, service.ErrNotLiked):
		c.JSON(http.StatusConflict, gin.H{"error": "not liked"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "like operation failed"})
	}
}

func userIdentifier(c *gin.Context) string {
	if header := strings.TrimSpace(c.GetHeader("X-User-Id")); header != "" {
		return header
	}
	return c.ClientIP()
}
