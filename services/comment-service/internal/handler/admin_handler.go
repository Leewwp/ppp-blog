package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/comment-service/internal/middleware"
	"github.com/ppp-blog/comment-service/internal/model"
	"github.com/ppp-blog/comment-service/internal/queue"
	"github.com/ppp-blog/comment-service/internal/repository"
	"github.com/ppp-blog/comment-service/internal/service"
)

// AdminHandler 提供管理后台接口。
type AdminHandler struct {
	producer *queue.Producer
	likeSvc  *service.LikeService
	repo     *repository.CommentRepo
	metrics  *middleware.Metrics
}

// NewAdminHandler 创建管理处理器。
func NewAdminHandler(
	producer *queue.Producer,
	likeSvc *service.LikeService,
	repo *repository.CommentRepo,
	metrics *middleware.Metrics,
) *AdminHandler {
	return &AdminHandler{
		producer: producer,
		likeSvc:  likeSvc,
		repo:     repo,
		metrics:  metrics,
	}
}

// Approve 审核通过评论。
func (h *AdminHandler) Approve(c *gin.Context) {
	h.updateStatus(c, model.StatusApproved)
}

// Reject 审核拒绝评论。
func (h *AdminHandler) Reject(c *gin.Context) {
	h.updateStatus(c, model.StatusRejected)
}

// SyncLikes 手动触发点赞同步。
func (h *AdminHandler) SyncLikes(c *gin.Context) {
	synced, err := h.likeSvc.SyncDirtyLikes(c.Request.Context(), h.repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync likes failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"synced": synced})
}

func (h *AdminHandler) updateStatus(c *gin.Context, status int8) {
	commentID := strings.TrimSpace(c.Param("comment_id"))
	postID := strings.TrimSpace(c.Query("post_id"))
	if commentID == "" || postID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment_id and post_id are required"})
		return
	}

	event := model.CommentEvent{
		EventType: "update_status",
		Comment: &model.Comment{
			CommentID: commentID,
			PostID:    postID,
			Status:    status,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := h.producer.Publish(c.Request.Context(), event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "publish status event failed"})
		return
	}
	h.metrics.IncKafkaProduced()
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
