package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ppp-blog/comment-service/internal/cache"
	"github.com/ppp-blog/comment-service/internal/middleware"
	"github.com/ppp-blog/comment-service/internal/model"
	"github.com/ppp-blog/comment-service/internal/queue"
	"github.com/ppp-blog/comment-service/internal/repository"
)

// CommentService 封装评论相关业务逻辑。
type CommentService struct {
	repo         *repository.CommentRepo
	commentCache *cache.CommentCache
	hotCache     *cache.HotCommentCache
	producer     *queue.Producer
	metrics      *middleware.Metrics
	logger       *slog.Logger
}

// NewCommentService 创建评论服务。
func NewCommentService(
	repo *repository.CommentRepo,
	commentCache *cache.CommentCache,
	hotCache *cache.HotCommentCache,
	producer *queue.Producer,
	metrics *middleware.Metrics,
	logger *slog.Logger,
) *CommentService {
	return &CommentService{
		repo:         repo,
		commentCache: commentCache,
		hotCache:     hotCache,
		producer:     producer,
		metrics:      metrics,
		logger:       logger,
	}
}

// SubmitComment 提交评论事件并返回评论 ID。
func (s *CommentService) SubmitComment(
	ctx context.Context,
	req model.SubmitCommentRequest,
	clientIP string,
) (string, error) {
	now := time.Now()
	commentID := uuid.NewString()

	comment := &model.Comment{
		CommentID: commentID,
		PostID:    strings.TrimSpace(req.PostID),
		ParentID:  strings.TrimSpace(req.ParentID),
		Author:    strings.TrimSpace(req.Author),
		AuthorIP:  strings.TrimSpace(clientIP),
		Content:   strings.TrimSpace(req.Content),
		Status:    model.StatusPending,
		LikeCount: 0,
		IsHot:     false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	event := model.CommentEvent{
		EventType: "create",
		Comment:   comment,
		Timestamp: now.UnixMilli(),
	}
	if err := s.producer.Publish(ctx, event); err != nil {
		return "", fmt.Errorf("publish create comment event: %w", err)
	}
	s.metrics.IncKafkaProduced()

	if err := s.commentCache.SetDetail(ctx, comment); err != nil {
		s.logger.Warn("set pending comment cache failed", "error", err, "comment_id", commentID)
	}
	return commentID, nil
}

// ListComments 返回游标分页结果。
func (s *CommentService) ListComments(
	ctx context.Context,
	req model.CursorPageRequest,
) (*model.CursorPageResponse, error) {
	pageSize := normalizePageSize(req.PageSize)

	cached, err := s.commentCache.GetPage(ctx, req.PostID, req.Cursor)
	if err != nil {
		s.logger.Warn("get page cache failed", "error", err, "post_id", req.PostID)
	}
	if cached != nil {
		s.metrics.IncCacheHit("page")
		return cached, nil
	}
	s.metrics.IncCacheMiss("page")

	cursorTime, cursorID, err := repository.ParseCursor(req.Cursor)
	if err != nil {
		return nil, fmt.Errorf("parse cursor: %w", err)
	}

	comments, err := s.repo.FindByPostIDWithCursor(ctx, req.PostID, cursorTime, cursorID, pageSize+1)
	if err != nil {
		return nil, fmt.Errorf("query comments by cursor: %w", err)
	}

	hasMore := len(comments) > pageSize
	if hasMore {
		comments = comments[:pageSize]
	}

	responses := make([]model.CommentResponse, 0, len(comments))
	for _, comment := range comments {
		responses = append(responses, comment.ToResponse())
	}

	total, err := s.GetCommentCount(ctx, req.PostID)
	if err != nil {
		return nil, err
	}

	result := &model.CursorPageResponse{
		Comments: responses,
		HasMore:  hasMore,
		Total:    total,
	}
	if hasMore && len(comments) > 0 {
		last := comments[len(comments)-1]
		result.NextCursor = repository.BuildCursor(last.CreatedAt, last.ID)
	}

	if err := s.commentCache.SetPage(ctx, req.PostID, req.Cursor, result); err != nil {
		s.logger.Warn("set page cache failed", "error", err, "post_id", req.PostID)
	}
	return result, nil
}

// GetHotComments 返回文章热评。
func (s *CommentService) GetHotComments(
	ctx context.Context,
	postID string,
	topN int,
) ([]model.CommentResponse, error) {
	if topN <= 0 {
		topN = 10
	}

	hotItems, err := s.hotCache.GetTopN(ctx, postID, topN)
	if err != nil {
		return nil, fmt.Errorf("query hot comments: %w", err)
	}

	responses := make([]model.CommentResponse, 0, len(hotItems))
	for _, item := range hotItems {
		comment, err := s.getCommentDetail(ctx, item.CommentID, postID)
		if err != nil {
			s.logger.Warn("get hot comment detail failed", "comment_id", item.CommentID, "error", err)
			continue
		}
		if comment == nil || comment.Status != model.StatusApproved {
			continue
		}
		responses = append(responses, comment.ToResponse())
	}
	return responses, nil
}

// GetCommentCount 返回文章评论数。
func (s *CommentService) GetCommentCount(ctx context.Context, postID string) (int64, error) {
	count, hit, err := s.commentCache.GetCount(ctx, postID)
	if err != nil {
		s.logger.Warn("get comment count cache failed", "error", err, "post_id", postID)
	}
	if hit {
		s.metrics.IncCacheHit("count")
		return count, nil
	}
	s.metrics.IncCacheMiss("count")

	count, err = s.repo.CountByPostID(ctx, postID)
	if err != nil {
		return 0, fmt.Errorf("count comments from mysql: %w", err)
	}
	if err := s.commentCache.SetCount(ctx, postID, count); err != nil {
		s.logger.Warn("set comment count cache failed", "error", err, "post_id", postID)
	}
	return count, nil
}

func (s *CommentService) getCommentDetail(
	ctx context.Context,
	commentID string,
	postID string,
) (*model.Comment, error) {
	cached, err := s.commentCache.GetDetail(ctx, commentID)
	if err != nil {
		s.logger.Warn("get comment detail cache failed", "error", err, "comment_id", commentID)
	}
	if cached != nil {
		s.metrics.IncCacheHit("detail")
		return cached, nil
	}
	s.metrics.IncCacheMiss("detail")

	comment, err := s.repo.FindByCommentID(ctx, commentID, postID)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, nil
	}

	if err := s.commentCache.SetDetail(ctx, comment); err != nil {
		s.logger.Warn("set comment detail cache failed", "error", err, "comment_id", commentID)
	}
	return comment, nil
}

func normalizePageSize(pageSize int) int {
	if pageSize <= 0 {
		return 20
	}
	if pageSize > 50 {
		return 50
	}
	return pageSize
}
