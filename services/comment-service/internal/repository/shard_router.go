package repository

import (
	"fmt"
	"hash/crc32"
)

// ShardRouter 根据 post_id 将请求路由到分表。
type ShardRouter struct {
	shardCount int
}

// NewShardRouter 创建分片路由器。
func NewShardRouter(shardCount int) *ShardRouter {
	if shardCount <= 0 {
		shardCount = 1
	}
	return &ShardRouter{shardCount: shardCount}
}

// TableName 返回 post_id 对应分表名称。
func (r *ShardRouter) TableName(postID string) string {
	return fmt.Sprintf("comment_shard_%d", r.ShardIndex(postID))
}

// ShardIndex 计算 post_id 对应分片下标。
func (r *ShardRouter) ShardIndex(postID string) int {
	hash := crc32.ChecksumIEEE([]byte(postID))
	return int(hash % uint32(r.shardCount))
}

// AllTableNames 返回所有分表名称。
func (r *ShardRouter) AllTableNames() []string {
	tables := make([]string, 0, r.shardCount)
	for i := 0; i < r.shardCount; i++ {
		tables = append(tables, fmt.Sprintf("comment_shard_%d", i))
	}
	return tables
}
