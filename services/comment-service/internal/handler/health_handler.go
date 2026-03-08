package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

// HealthHandler 提供 /health 健康检查接口。
type HealthHandler struct {
	db           *sql.DB
	rdb          *redis.Client
	kafkaBrokers []string
}

// NewHealthHandler 创建健康检查处理器。
func NewHealthHandler(db *sql.DB, rdb *redis.Client, kafkaBrokers []string) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb, kafkaBrokers: kafkaBrokers}
}

// Health 检查 MySQL、Redis、Kafka 连通性。
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	status := "ok"
	details := map[string]string{}

	if err := h.db.PingContext(ctx); err != nil {
		status = "degraded"
		details["mysql"] = err.Error()
	} else {
		details["mysql"] = "healthy"
	}

	if err := h.rdb.Ping(ctx).Err(); err != nil {
		status = "degraded"
		details["redis"] = err.Error()
	} else {
		details["redis"] = "healthy"
	}

	if err := h.checkKafka(ctx); err != nil {
		status = "degraded"
		details["kafka"] = err.Error()
	} else {
		details["kafka"] = "healthy"
	}

	code := http.StatusOK
	if status != "ok" {
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, gin.H{"status": status, "details": details})
}

func (h *HealthHandler) checkKafka(ctx context.Context) error {
	if len(h.kafkaBrokers) == 0 {
		return nil
	}
	conn, err := kafka.DialContext(ctx, "tcp", h.kafkaBrokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Brokers()
	return err
}
