package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ppp-blog/comment-service/internal/model"
	"github.com/segmentio/kafka-go"
)

// Producer 负责发送评论事件到 Kafka。
type Producer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

// NewProducer 创建 Kafka 生产者。
func NewProducer(brokers []string, topic string, logger *slog.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		BatchTimeout:           50 * time.Millisecond,
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
	}
	return &Producer{writer: writer, logger: logger}
}

// Publish 发布评论事件。
func (p *Producer) Publish(ctx context.Context, event model.CommentEvent) error {
	if event.Comment == nil || event.Comment.PostID == "" {
		return fmt.Errorf("invalid comment event payload")
	}

	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal comment event: %w", err)
	}

	message := kafka.Message{
		Key:   []byte(event.Comment.PostID),
		Value: value,
		Time:  time.Now(),
	}
	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("publish kafka message: %w", err)
	}

	p.logger.Debug(
		"kafka message published",
		"event_type", event.EventType,
		"comment_id", event.Comment.CommentID,
		"post_id", event.Comment.PostID,
	)
	return nil
}

// Close 关闭生产者。
func (p *Producer) Close() error {
	return p.writer.Close()
}
