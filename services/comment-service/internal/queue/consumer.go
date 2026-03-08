package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ppp-blog/comment-service/internal/cache"
	"github.com/ppp-blog/comment-service/internal/middleware"
	"github.com/ppp-blog/comment-service/internal/model"
	"github.com/ppp-blog/comment-service/internal/repository"
	"github.com/segmentio/kafka-go"
)

const maxConsumeRetry = 3

// Consumer 消费 Kafka 中的评论事件。
type Consumer struct {
	reader       *kafka.Reader
	repo         *repository.CommentRepo
	commentCache *cache.CommentCache
	hotCache     *cache.HotCommentCache
	metrics      *middleware.Metrics
	logger       *slog.Logger
}

// NewConsumer 创建 Kafka 消费者。
func NewConsumer(
	brokers []string,
	topic string,
	group string,
	repo *repository.CommentRepo,
	commentCache *cache.CommentCache,
	hotCache *cache.HotCommentCache,
	metrics *middleware.Metrics,
	logger *slog.Logger,
) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        group,
		MinBytes:       1 * 1024,
		MaxBytes:       10 * 1024 * 1024,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	return &Consumer{
		reader:       reader,
		repo:         repo,
		commentCache: commentCache,
		hotCache:     hotCache,
		metrics:      metrics,
		logger:       logger,
	}
}

// Start 启动消费者主循环。
func (c *Consumer) Start(ctx context.Context) error {
	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.metrics.IncKafkaConsumeError()
			c.logger.Error("fetch kafka message failed", "error", err)
			continue
		}

		if err := c.handleWithRetry(ctx, message); err != nil {
			c.metrics.IncKafkaConsumeError()
			c.logger.Error("handle kafka message failed", "error", err, "offset", message.Offset)
		}

		if err := c.reader.CommitMessages(ctx, message); err != nil {
			c.metrics.IncKafkaConsumeError()
			c.logger.Error("commit kafka message failed", "error", err, "offset", message.Offset)
		}
	}
}

// Close 关闭消费者。
func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) handleWithRetry(ctx context.Context, message kafka.Message) error {
	var lastErr error
	for attempt := 1; attempt <= maxConsumeRetry; attempt++ {
		if err := c.handleMessage(ctx, message); err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
			continue
		}
		return nil
	}
	return fmt.Errorf("retry exhausted: %w", lastErr)
}

func (c *Consumer) handleMessage(ctx context.Context, message kafka.Message) error {
	var event model.CommentEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("unmarshal comment event: %w", err)
	}
	if event.Comment == nil {
		return fmt.Errorf("comment event payload is nil")
	}

	switch event.EventType {
	case "create":
		if err := c.onCreate(ctx, event.Comment); err != nil {
			return err
		}
	case "update_status":
		if err := c.onStatusUpdate(ctx, event.Comment); err != nil {
			return err
		}
	case "like":
		if err := c.onLikeDelta(ctx, event.Comment, 1); err != nil {
			return err
		}
	case "unlike":
		if err := c.onLikeDelta(ctx, event.Comment, -1); err != nil {
			return err
		}
	default:
		c.logger.Warn("skip unknown event type", "event_type", event.EventType)
		return nil
	}

	c.metrics.IncKafkaConsumed(event.EventType)
	return nil
}

func (c *Consumer) onCreate(ctx context.Context, comment *model.Comment) error {
	if err := c.repo.Insert(ctx, comment); err != nil {
		return fmt.Errorf("insert comment in create event: %w", err)
	}
	c.commentCache.InvalidatePost(ctx, comment.PostID)
	c.commentCache.InvalidateDetail(ctx, comment.CommentID)
	return nil
}

func (c *Consumer) onStatusUpdate(ctx context.Context, comment *model.Comment) error {
	if err := c.repo.UpdateStatus(ctx, comment.CommentID, comment.PostID, comment.Status); err != nil {
		return fmt.Errorf("update comment status: %w", err)
	}
	c.commentCache.InvalidatePost(ctx, comment.PostID)
	c.commentCache.InvalidateDetail(ctx, comment.CommentID)

	if comment.Status != model.StatusApproved {
		if err := c.hotCache.RemoveMember(ctx, comment.PostID, comment.CommentID); err != nil {
			c.logger.Warn("remove hot comment failed", "error", err, "comment_id", comment.CommentID)
		}
	}
	return nil
}

func (c *Consumer) onLikeDelta(ctx context.Context, comment *model.Comment, delta int) error {
	if err := c.repo.IncrLikeCount(ctx, comment.CommentID, comment.PostID, delta); err != nil {
		return fmt.Errorf("update like count by delta: %w", err)
	}

	latest, err := c.repo.FindByCommentID(ctx, comment.CommentID, comment.PostID)
	if err != nil {
		return fmt.Errorf("load comment after like update: %w", err)
	}
	if latest == nil {
		return nil
	}

	if err := c.commentCache.SetDetail(ctx, latest); err != nil {
		c.logger.Warn("set comment detail cache failed", "error", err, "comment_id", latest.CommentID)
	}

	score := cache.CalculateScore(latest.LikeCount, 0, latest.CreatedAt)
	if err := c.hotCache.UpdateScore(ctx, latest.PostID, latest.CommentID, score); err != nil {
		c.logger.Warn("update hot score failed", "error", err, "comment_id", latest.CommentID)
	}
	return nil
}
