package model

import "time"

const (
	// StatusPending 评论待审核。
	StatusPending int8 = 0
	// StatusApproved 评论审核通过。
	StatusApproved int8 = 1
	// StatusRejected 评论审核拒绝。
	StatusRejected int8 = 2
	// StatusDeleted 评论被删除。
	StatusDeleted int8 = 3
)

// Comment 对应 comment_shard_N 表。
type Comment struct {
	ID        int64     `json:"id"`
	CommentID string    `json:"comment_id"`
	PostID    string    `json:"post_id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Author    string    `json:"author"`
	AuthorIP  string    `json:"author_ip,omitempty"`
	Content   string    `json:"content"`
	Status    int8      `json:"status"`
	LikeCount int       `json:"like_count"`
	IsHot     bool      `json:"is_hot"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SubmitCommentRequest 提交评论请求。
type SubmitCommentRequest struct {
	PostID   string `json:"post_id" binding:"required"`
	ParentID string `json:"parent_id"`
	Author   string `json:"author" binding:"required,max=128"`
	Content  string `json:"content" binding:"required,max=2000"`
}

// CommentResponse 评论响应。
type CommentResponse struct {
	CommentID string    `json:"comment_id"`
	PostID    string    `json:"post_id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	LikeCount int       `json:"like_count"`
	IsHot     bool      `json:"is_hot"`
	CreatedAt time.Time `json:"created_at"`
}

// CursorPageRequest 游标分页请求。
type CursorPageRequest struct {
	PostID   string `form:"post_id" binding:"required"`
	Cursor   string `form:"cursor"`
	PageSize int    `form:"page_size"`
}

// CursorPageResponse 游标分页响应。
type CursorPageResponse struct {
	Comments   []CommentResponse `json:"comments"`
	NextCursor string            `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
	Total      int64             `json:"total"`
}

// HotCommentRequest 热评请求。
type HotCommentRequest struct {
	PostID string `form:"post_id" binding:"required"`
	TopN   int    `form:"top_n"`
}

// CommentEvent Kafka 评论事件。
type CommentEvent struct {
	EventType string   `json:"event_type"`
	Comment   *Comment `json:"comment"`
	Timestamp int64    `json:"timestamp"`
}

// LikeRequest 点赞请求。
type LikeRequest struct {
	PostID string `json:"post_id" binding:"required"`
}

// ToResponse 将实体转换为响应 DTO。
func (c Comment) ToResponse() CommentResponse {
	return CommentResponse{
		CommentID: c.CommentID,
		PostID:    c.PostID,
		ParentID:  c.ParentID,
		Author:    c.Author,
		Content:   c.Content,
		LikeCount: c.LikeCount,
		IsHot:     c.IsHot,
		CreatedAt: c.CreatedAt,
	}
}
