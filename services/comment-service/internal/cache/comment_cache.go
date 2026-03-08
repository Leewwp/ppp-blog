package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ppp-blog/comment-service/internal/model"
	"github.com/redis/go-redis/v9"
)

const (
	pageKeyPrefix   = "comment:page:"
	countKeyPrefix  = "comment:count:"
	detailKeyPrefix = "comment:detail:"

	pageTTL   = 30 * time.Second
	countTTL  = 60 * time.Second
	detailTTL = 5 * time.Minute
)

// CommentCache 管理评论分页、总数和详情缓存。
type CommentCache struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// NewCommentCache 创建评论缓存。
func NewCommentCache(rdb *redis.Client, logger *slog.Logger) *CommentCache {
	return &CommentCache{rdb: rdb, logger: logger}
}

// GetPage 读取分页缓存。
func (c *CommentCache) GetPage(
	ctx context.Context,
	postID string,
	cursor string,
) (*model.CursorPageResponse, error) {
	key := c.pageKey(postID, cursor)
	payload, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get page cache: %w", err)
	}

	var response model.CursorPageResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, fmt.Errorf("unmarshal page cache: %w", err)
	}
	return &response, nil
}

// SetPage 写入分页缓存。
func (c *CommentCache) SetPage(
	ctx context.Context,
	postID string,
	cursor string,
	response *model.CursorPageResponse,
) error {
	key := c.pageKey(postID, cursor)
	payload, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal page response: %w", err)
	}
	if err := c.rdb.Set(ctx, key, payload, pageTTL).Err(); err != nil {
		return fmt.Errorf("set page cache: %w", err)
	}
	return nil
}

// InvalidatePost 清理指定文章相关分页和计数缓存。
func (c *CommentCache) InvalidatePost(ctx context.Context, postID string) {
	pattern := fmt.Sprintf("%s%s:*", pageKeyPrefix, postID)
	c.scanAndDelete(ctx, pattern)

	if err := c.rdb.Del(ctx, countKeyPrefix+postID).Err(); err != nil {
		c.logger.Warn("delete comment count cache failed", "post_id", postID, "error", err)
	}
}

// GetCount 获取文章评论总数缓存。
func (c *CommentCache) GetCount(ctx context.Context, postID string) (int64, bool, error) {
	count, err := c.rdb.Get(ctx, countKeyPrefix+postID).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("get comment count cache: %w", err)
	}
	return count, true, nil
}

// SetCount 设置文章评论总数缓存。
func (c *CommentCache) SetCount(ctx context.Context, postID string, count int64) error {
	if err := c.rdb.Set(ctx, countKeyPrefix+postID, count, countTTL).Err(); err != nil {
		return fmt.Errorf("set comment count cache: %w", err)
	}
	return nil
}

// GetDetail 获取评论详情缓存。
func (c *CommentCache) GetDetail(ctx context.Context, commentID string) (*model.Comment, error) {
	payload, err := c.rdb.Get(ctx, detailKeyPrefix+commentID).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get comment detail cache: %w", err)
	}

	var comment model.Comment
	if err := json.Unmarshal(payload, &comment); err != nil {
		return nil, fmt.Errorf("unmarshal comment detail cache: %w", err)
	}
	return &comment, nil
}

// SetDetail 设置评论详情缓存。
func (c *CommentCache) SetDetail(ctx context.Context, comment *model.Comment) error {
	payload, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("marshal comment detail: %w", err)
	}
	if err := c.rdb.Set(ctx, detailKeyPrefix+comment.CommentID, payload, detailTTL).Err(); err != nil {
		return fmt.Errorf("set comment detail cache: %w", err)
	}
	return nil
}

// InvalidateDetail 删除评论详情缓存。
func (c *CommentCache) InvalidateDetail(ctx context.Context, commentID string) {
	if err := c.rdb.Del(ctx, detailKeyPrefix+commentID).Err(); err != nil {
		c.logger.Warn("delete comment detail cache failed", "comment_id", commentID, "error", err)
	}
}

func (c *CommentCache) pageKey(postID string, cursor string) string {
	hash := md5.Sum([]byte(cursor))
	return fmt.Sprintf("%s%s:%s", pageKeyPrefix, postID, hex.EncodeToString(hash[:]))
}

func (c *CommentCache) scanAndDelete(ctx context.Context, pattern string) {
	var cursor uint64
	for {
		keys, nextCursor, err := c.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			c.logger.Warn("scan cache keys failed", "pattern", pattern, "error", err)
			return
		}
		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				c.logger.Warn("delete cache keys failed", "pattern", pattern, "error", err)
			}
		}
		if nextCursor == 0 {
			return
		}
		cursor = nextCursor
	}
}
