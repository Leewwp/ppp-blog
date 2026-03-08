package cache

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

const hotKeyPrefix = "hot:comments:"

// HotCommentItem 表示热评排行中的单条记录。
type HotCommentItem struct {
	CommentID string  `json:"comment_id"`
	Score     float64 `json:"score"`
}

// HotCommentCache 管理热评榜单缓存。
type HotCommentCache struct {
	rdb *redis.Client
}

// NewHotCommentCache 创建热评缓存。
func NewHotCommentCache(rdb *redis.Client) *HotCommentCache {
	return &HotCommentCache{rdb: rdb}
}

// CalculateScore 按指定公式计算热评分数。
func CalculateScore(likeCount int, replyCount int, createdAt time.Time) float64 {
	days := time.Since(createdAt).Hours() / 24
	timeBonus := math.Max(0, 7-days) * 2
	return float64(likeCount*3+replyCount*5) + timeBonus
}

// UpdateScore 更新评论热评分数。
func (h *HotCommentCache) UpdateScore(
	ctx context.Context,
	postID string,
	commentID string,
	score float64,
) error {
	if err := h.rdb.ZAdd(ctx, hotKeyPrefix+postID, redis.Z{Member: commentID, Score: score}).Err(); err != nil {
		return fmt.Errorf("update hot comment score: %w", err)
	}
	return nil
}

// GetTopN 获取热评榜前 N 条（含分数）。
func (h *HotCommentCache) GetTopN(ctx context.Context, postID string, n int) ([]HotCommentItem, error) {
	if n <= 0 {
		return []HotCommentItem{}, nil
	}
	items, err := h.rdb.ZRevRangeWithScores(ctx, hotKeyPrefix+postID, 0, int64(n-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("get hot comments top n: %w", err)
	}

	result := make([]HotCommentItem, 0, len(items))
	for _, item := range items {
		commentID, ok := item.Member.(string)
		if !ok || commentID == "" {
			continue
		}
		result = append(result, HotCommentItem{CommentID: commentID, Score: item.Score})
	}
	return result, nil
}

// RemoveMember 移除热评榜中的评论。
func (h *HotCommentCache) RemoveMember(ctx context.Context, postID string, commentID string) error {
	if err := h.rdb.ZRem(ctx, hotKeyPrefix+postID, commentID).Err(); err != nil {
		return fmt.Errorf("remove hot comment member: %w", err)
	}
	return nil
}
