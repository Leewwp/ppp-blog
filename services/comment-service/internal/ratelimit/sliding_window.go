package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// SlidingWindowLimiter 提供基于 Redis 的滑动窗口限流能力。
type SlidingWindowLimiter struct {
	rdb *redis.Client
}

// NewSlidingWindowLimiter 创建限流器。
func NewSlidingWindowLimiter(rdb *redis.Client) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{rdb: rdb}
}

// Allow 判断 key 在窗口内是否允许通过。
func (l *SlidingWindowLimiter) Allow(
	ctx context.Context,
	key string,
	windowSeconds int,
	maxRequests int,
) (bool, int, error) {
	result, err := l.rdb.Eval(
		ctx,
		slidingWindowScript,
		[]string{key},
		time.Now().UnixMilli(),
		windowSeconds,
		maxRequests,
	).Result()
	if err != nil {
		return false, 0, fmt.Errorf("execute sliding window script: %w", err)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return false, 0, fmt.Errorf("unexpected script result: %v", result)
	}

	allowed, err := toInt64(values[0])
	if err != nil {
		return false, 0, err
	}
	remaining, err := toInt64(values[1])
	if err != nil {
		return false, 0, err
	}

	return allowed == 1, int(remaining), nil
}

func toInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse int result: %w", err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unexpected value type: %T", value)
	}
}
