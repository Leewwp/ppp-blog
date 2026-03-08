package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/comment-service/internal/config"
	"github.com/ppp-blog/comment-service/internal/ratelimit"
)

// RateLimit 在评论接口上执行 IP + 用户双维度限流。
func RateLimit(
	limiter *ratelimit.SlidingWindowLimiter,
	cfg config.RateLimitConfig,
	metrics *Metrics,
	logger *slog.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		ipAllowed, ipRemaining, ipErr := checkIPLimit(c, limiter, cfg)
		if ipErr != nil {
			logger.Error("check ip limit failed", "error", ipErr, "ip", c.ClientIP())
			c.Next()
			return
		}
		if !ipAllowed {
			reject(c, metrics, "ip", cfg.IPWindowSeconds, "IP rate limit exceeded")
			return
		}

		remaining := ipRemaining
		if isCommentSubmit(c) {
			author, err := extractAuthor(c)
			if err != nil {
				logger.Warn("extract author for rate limit failed", "error", err)
			}
			if author == "" {
				author = c.ClientIP()
			}
			userKey := fmt.Sprintf("rate:user:%s", author)
			allowed, userRemaining, userErr := limiter.Allow(c.Request.Context(), userKey, cfg.WindowSeconds, cfg.MaxRequests)
			if userErr != nil {
				logger.Error("check user limit failed", "error", userErr, "author", author)
				c.Next()
				return
			}
			if !allowed {
				reject(c, metrics, "user", cfg.WindowSeconds, "User rate limit exceeded")
				return
			}
			remaining = minInt(remaining, userRemaining)
		}

		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Next()
	}
}

func checkIPLimit(
	c *gin.Context,
	limiter *ratelimit.SlidingWindowLimiter,
	cfg config.RateLimitConfig,
) (bool, int, error) {
	ipKey := fmt.Sprintf("rate:ip:%s", c.ClientIP())
	allowed, remaining, err := limiter.Allow(c.Request.Context(), ipKey, cfg.IPWindowSeconds, cfg.IPMaxRequests)
	if err != nil {
		return false, 0, err
	}
	return allowed, remaining, nil
}

func reject(c *gin.Context, metrics *Metrics, dimension string, retryAfter int, message string) {
	metrics.IncRateLimitRejected(dimension)
	c.Header("Retry-After", strconv.Itoa(retryAfter))
	c.Header("X-RateLimit-Remaining", "0")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": message})
}

func isCommentSubmit(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	fullPath := c.FullPath()
	return fullPath == "/api/v1/comments"
}

func extractAuthor(c *gin.Context) (string, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "", err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	if len(body) == 0 {
		return "", nil
	}

	var payload struct {
		Author string `json:"author"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.Author), nil
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
