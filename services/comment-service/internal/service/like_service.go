package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ppp-blog/comment-service/internal/cache"
	"github.com/ppp-blog/comment-service/internal/middleware"
	"github.com/ppp-blog/comment-service/internal/model"
	"github.com/ppp-blog/comment-service/internal/queue"
	"github.com/ppp-blog/comment-service/internal/repository"
)

var (
	// ErrAlreadyLiked 表示用户重复点赞。
	ErrAlreadyLiked = errors.New("already liked")
	// ErrNotLiked 表示用户尚未点赞。
	ErrNotLiked = errors.New("not liked")
)

// LikeService 处理点赞业务与异步同步。
type LikeService struct {
	repo         *repository.CommentRepo
	commentCache *cache.CommentCache
	likeCache    *cache.LikeCache
	hotCache     *cache.HotCommentCache
	producer     *queue.Producer
	metrics      *middleware.Metrics
	logger       *slog.Logger
}

// NewLikeService 创建点赞服务。
func NewLikeService(
	repo *repository.CommentRepo,
	commentCache *cache.CommentCache,
	likeCache *cache.LikeCache,
	hotCache *cache.HotCommentCache,
	producer *queue.Producer,
	metrics *middleware.Metrics,
	logger *slog.Logger,
) *LikeService {
	return &LikeService{
		repo:         repo,
		commentCache: commentCache,
		likeCache:    likeCache,
		hotCache:     hotCache,
		producer:     producer,
		metrics:      metrics,
		logger:       logger,
	}
}

// Like 执行点赞。
func (s *LikeService) Like(
	ctx context.Context,
	commentID string,
	postID string,
	userIdentifier string,
) error {
	changed, count, err := s.likeCache.IncrLike(ctx, commentID, userIdentifier)
	if err != nil {
		return fmt.Errorf("increment like cache: %w", err)
	}
	if !changed {
		return ErrAlreadyLiked
	}

	if err := s.publishLikeEvent(ctx, "like", commentID, postID, int(count)); err != nil {
		s.logger.Error("publish like event failed", "error", err, "comment_id", commentID)
	}
	if err := s.likeCache.MarkDirty(ctx, buildDirtyMarker(postID, commentID)); err != nil {
		s.logger.Warn("mark like dirty failed", "error", err, "comment_id", commentID)
	}
	if err := s.updateHotScore(ctx, commentID, postID); err != nil {
		s.logger.Warn("update hot score failed", "error", err, "comment_id", commentID)
	}
	return nil
}

// Unlike 执行取消点赞。
func (s *LikeService) Unlike(
	ctx context.Context,
	commentID string,
	postID string,
	userIdentifier string,
) error {
	changed, count, err := s.likeCache.DecrLike(ctx, commentID, userIdentifier)
	if err != nil {
		return fmt.Errorf("decrement like cache: %w", err)
	}
	if !changed {
		return ErrNotLiked
	}

	if err := s.publishLikeEvent(ctx, "unlike", commentID, postID, int(count)); err != nil {
		s.logger.Error("publish unlike event failed", "error", err, "comment_id", commentID)
	}
	if err := s.likeCache.MarkDirty(ctx, buildDirtyMarker(postID, commentID)); err != nil {
		s.logger.Warn("mark unlike dirty failed", "error", err, "comment_id", commentID)
	}
	if err := s.updateHotScore(ctx, commentID, postID); err != nil {
		s.logger.Warn("update hot score failed", "error", err, "comment_id", commentID)
	}
	return nil
}

// HasLiked 查询当前用户是否已点赞。
func (s *LikeService) HasLiked(
	ctx context.Context,
	commentID string,
	userIdentifier string,
) (bool, error) {
	liked, err := s.likeCache.HasLiked(ctx, commentID, userIdentifier)
	if err != nil {
		return false, fmt.Errorf("query like state: %w", err)
	}
	return liked, nil
}

// StartSyncLoop 启动点赞数据定时同步任务。
func (s *LikeService) StartSyncLoop(ctx context.Context, repo *repository.CommentRepo) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.SyncDirtyLikes(ctx, repo); err != nil {
				s.logger.Error("sync dirty likes failed", "error", err)
			}
		}
	}
}

// SyncDirtyLikes 立即同步一批脏点赞数据。
func (s *LikeService) SyncDirtyLikes(ctx context.Context, repo *repository.CommentRepo) (int, error) {
	markers, err := s.likeCache.PopDirtyBatch(ctx)
	if err != nil {
		return 0, err
	}
	if len(markers) == 0 {
		return 0, nil
	}

	success := 0
	for _, marker := range markers {
		postID, commentID, parseErr := parseDirtyMarker(marker)
		if parseErr != nil {
			s.logger.Warn("skip invalid dirty marker", "marker", marker, "error", parseErr)
			continue
		}

		count, countErr := s.likeCache.GetLikeCount(ctx, commentID)
		if countErr != nil {
			s.requeueDirty(ctx, marker, countErr)
			continue
		}

		if syncErr := repo.SyncLikeCount(ctx, commentID, postID, count); syncErr != nil {
			s.requeueDirty(ctx, marker, syncErr)
			continue
		}

		s.commentCache.InvalidateDetail(ctx, commentID)
		success++
	}
	return success, nil
}

func (s *LikeService) publishLikeEvent(
	ctx context.Context,
	eventType string,
	commentID string,
	postID string,
	likeCount int,
) error {
	event := model.CommentEvent{
		EventType: eventType,
		Comment: &model.Comment{
			CommentID: commentID,
			PostID:    postID,
			LikeCount: likeCount,
			CreatedAt: time.Now(),
		},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := s.producer.Publish(ctx, event); err != nil {
		return fmt.Errorf("publish %s event: %w", eventType, err)
	}
	s.metrics.IncKafkaProduced()
	return nil
}

func (s *LikeService) updateHotScore(ctx context.Context, commentID string, postID string) error {
	comment, err := s.repo.FindByCommentID(ctx, commentID, postID)
	if err != nil {
		return err
	}
	if comment == nil {
		return nil
	}

	score := cache.CalculateScore(comment.LikeCount, 0, comment.CreatedAt)
	if err := s.hotCache.UpdateScore(ctx, postID, commentID, score); err != nil {
		return err
	}
	return nil
}

func (s *LikeService) requeueDirty(ctx context.Context, marker string, reason error) {
	s.logger.Warn("sync dirty like failed and requeue", "marker", marker, "error", reason)
	if err := s.likeCache.MarkDirty(ctx, marker); err != nil {
		s.logger.Error("requeue dirty like failed", "marker", marker, "error", err)
	}
}

func buildDirtyMarker(postID string, commentID string) string {
	return strings.TrimSpace(postID) + "|" + strings.TrimSpace(commentID)
}

func parseDirtyMarker(marker string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(marker), "|")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid marker format")
	}
	postID := strings.TrimSpace(parts[0])
	commentID := strings.TrimSpace(parts[1])
	if postID == "" || commentID == "" {
		return "", "", fmt.Errorf("empty post_id or comment_id in marker")
	}
	return postID, commentID, nil
}
