package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

const (
	likeCountPrefix = "like:count:"
	likeUsersPrefix = "like:users:"
	likeSyncPending = "like:sync:pending"
	likeSyncBatch   = 100
)

const incrLikeScript = `
local user_key = KEYS[1]
local count_key = KEYS[2]
local user = ARGV[1]
local added = redis.call('SADD', user_key, user)
if added == 1 then
    local count = redis.call('INCR', count_key)
    return {1, count}
end
local current = redis.call('GET', count_key)
if not current then
    current = '0'
end
return {0, tonumber(current)}
`

const decrLikeScript = `
local user_key = KEYS[1]
local count_key = KEYS[2]
local user = ARGV[1]
local removed = redis.call('SREM', user_key, user)
if removed == 1 then
    local count = redis.call('DECR', count_key)
    if count < 0 then
        redis.call('SET', count_key, 0)
        count = 0
    end
    return {1, count}
end
local current = redis.call('GET', count_key)
if not current then
    current = '0'
end
return {0, tonumber(current)}
`

// LikeCache 管理点赞相关缓存。
type LikeCache struct {
	rdb *redis.Client
}

// NewLikeCache 创建点赞缓存组件。
func NewLikeCache(rdb *redis.Client) *LikeCache {
	return &LikeCache{rdb: rdb}
}

// IncrLike 点赞并返回是否成功新增。
func (l *LikeCache) IncrLike(
	ctx context.Context,
	commentID string,
	userIdentifier string,
) (bool, int64, error) {
	return l.evalLikeScript(ctx, incrLikeScript, commentID, userIdentifier)
}

// DecrLike 取消点赞并返回是否成功取消。
func (l *LikeCache) DecrLike(
	ctx context.Context,
	commentID string,
	userIdentifier string,
) (bool, int64, error) {
	return l.evalLikeScript(ctx, decrLikeScript, commentID, userIdentifier)
}

// HasLiked 查询用户是否已点赞。
func (l *LikeCache) HasLiked(ctx context.Context, commentID string, userIdentifier string) (bool, error) {
	liked, err := l.rdb.SIsMember(ctx, likeUsersPrefix+commentID, userIdentifier).Result()
	if err != nil {
		return false, fmt.Errorf("check like state: %w", err)
	}
	return liked, nil
}

// GetLikeCount 获取当前点赞数。
func (l *LikeCache) GetLikeCount(ctx context.Context, commentID string) (int64, error) {
	count, err := l.rdb.Get(ctx, likeCountPrefix+commentID).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("get like count: %w", err)
	}
	return count, nil
}

// MarkDirty 将脏数据标记加入同步集合。
func (l *LikeCache) MarkDirty(ctx context.Context, marker string) error {
	if err := l.rdb.SAdd(ctx, likeSyncPending, marker).Err(); err != nil {
		return fmt.Errorf("mark like dirty: %w", err)
	}
	return nil
}

// PopDirtyBatch 批量弹出待同步标记。
func (l *LikeCache) PopDirtyBatch(ctx context.Context) ([]string, error) {
	markers, err := l.rdb.SPopN(ctx, likeSyncPending, likeSyncBatch).Result()
	if err != nil {
		if err == redis.Nil {
			return []string{}, nil
		}
		return nil, fmt.Errorf("pop dirty batch: %w", err)
	}
	return markers, nil
}

func (l *LikeCache) evalLikeScript(
	ctx context.Context,
	script string,
	commentID string,
	userIdentifier string,
) (bool, int64, error) {
	result, err := l.rdb.Eval(
		ctx,
		script,
		[]string{likeUsersPrefix + commentID, likeCountPrefix + commentID},
		userIdentifier,
	).Result()
	if err != nil {
		return false, 0, fmt.Errorf("execute like script: %w", err)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return false, 0, fmt.Errorf("unexpected like script result: %v", result)
	}

	changed, err := parseInt64(values[0])
	if err != nil {
		return false, 0, err
	}
	count, err := parseInt64(values[1])
	if err != nil {
		return false, 0, err
	}
	return changed == 1, count, nil
}

func parseInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse int64: %w", err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unexpected type for int64: %T", value)
	}
}
