package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/comment-service/internal/model"
	"github.com/ppp-blog/comment-service/internal/service"
)

// CommentHandler 提供评论读写接口。
type CommentHandler struct {
	svc *service.CommentService
}

// NewCommentHandler 创建评论处理器。
func NewCommentHandler(svc *service.CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

// Submit 提交评论并返回 202。
func (h *CommentHandler) Submit(c *gin.Context) {
	var req model.SubmitCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.PostID = strings.TrimSpace(req.PostID)
	req.ParentID = strings.TrimSpace(req.ParentID)
	req.Author = strings.TrimSpace(req.Author)
	req.Content = strings.TrimSpace(req.Content)
	if req.PostID == "" || req.Author == "" || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post_id, author and content are required"})
		return
	}

	commentID, err := h.svc.SubmitComment(c.Request.Context(), req, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "submit comment failed"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"comment_id": commentID,
		"status":     "pending",
	})
}

// List 游标分页查询评论。
func (h *CommentHandler) List(c *gin.Context) {
	var req model.CursorPageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.svc.ListComments(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

// Hot 查询热评。
func (h *CommentHandler) Hot(c *gin.Context) {
	var req model.HotCommentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comments, err := h.svc.GetHotComments(c.Request.Context(), strings.TrimSpace(req.PostID), req.TopN)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get hot comments failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

// Count 查询评论总数。
func (h *CommentHandler) Count(c *gin.Context) {
	postID := strings.TrimSpace(c.Query("post_id"))
	if postID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post_id is required"})
		return
	}

	count, err := h.svc.GetCommentCount(c.Request.Context(), postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get comment count failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"post_id": postID, "count": count})
}
