package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ppp-blog/comment-service/internal/model"
)

// CommentRepo 负责评论分表读写。
type CommentRepo struct {
	db     *sql.DB
	router *ShardRouter
}

// NewCommentRepo 创建评论仓储。
func NewCommentRepo(db *sql.DB, router *ShardRouter) *CommentRepo {
	return &CommentRepo{db: db, router: router}
}

// Insert 向分表插入评论。
func (r *CommentRepo) Insert(ctx context.Context, comment *model.Comment) error {
	table := r.router.TableName(comment.PostID)
	query := fmt.Sprintf(`INSERT INTO %s
	(comment_id, post_id, parent_id, author, author_ip, content, status, like_count, is_hot, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, table)

	result, err := r.db.ExecContext(
		ctx,
		query,
		comment.CommentID,
		comment.PostID,
		nullableString(comment.ParentID),
		comment.Author,
		nullableString(comment.AuthorIP),
		comment.Content,
		comment.Status,
		comment.LikeCount,
		boolToTinyInt(comment.IsHot),
		comment.CreatedAt,
		comment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}

	insertID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("fetch inserted id: %w", err)
	}
	comment.ID = insertID
	return nil
}

// FindByPostIDWithCursor 按游标查询已通过评论。
func (r *CommentRepo) FindByPostIDWithCursor(
	ctx context.Context,
	postID string,
	cursorTime time.Time,
	cursorID int64,
	pageSize int,
) ([]model.Comment, error) {
	table := r.router.TableName(postID)
	if pageSize <= 0 {
		pageSize = 20
	}

	query := fmt.Sprintf(`SELECT id, comment_id, post_id, parent_id, author, author_ip, content,
	status, like_count, is_hot, created_at, updated_at
	FROM %s
	WHERE post_id = ? AND status = 1
	ORDER BY created_at DESC, id DESC
	LIMIT ?`, table)
	args := []interface{}{postID, pageSize}

	if cursorID > 0 {
		query = fmt.Sprintf(`SELECT id, comment_id, post_id, parent_id, author, author_ip, content,
		status, like_count, is_hot, created_at, updated_at
		FROM %s
		WHERE post_id = ? AND status = 1
		AND (created_at < ? OR (created_at = ? AND id < ?))
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, table)
		args = []interface{}{postID, cursorTime, cursorTime, cursorID, pageSize}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find comments by cursor: %w", err)
	}
	defer rows.Close()

	return scanComments(rows)
}

// CountByPostID 返回文章已通过评论数量。
func (r *CommentRepo) CountByPostID(ctx context.Context, postID string) (int64, error) {
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE post_id = ? AND status = 1",
		r.router.TableName(postID),
	)

	var count int64
	if err := r.db.QueryRowContext(ctx, query, postID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count comments: %w", err)
	}
	return count, nil
}

// IncrLikeCount 按增量更新点赞数。
func (r *CommentRepo) IncrLikeCount(ctx context.Context, commentID, postID string, delta int) error {
	query := fmt.Sprintf(
		"UPDATE %s SET like_count = GREATEST(0, like_count + ?) WHERE comment_id = ?",
		r.router.TableName(postID),
	)
	if _, err := r.db.ExecContext(ctx, query, delta, commentID); err != nil {
		return fmt.Errorf("increment like count: %w", err)
	}
	return nil
}

// SyncLikeCount 将点赞数同步为绝对值。
func (r *CommentRepo) SyncLikeCount(ctx context.Context, commentID, postID string, likeCount int64) error {
	query := fmt.Sprintf(
		"UPDATE %s SET like_count = ? WHERE comment_id = ?",
		r.router.TableName(postID),
	)
	if _, err := r.db.ExecContext(ctx, query, likeCount, commentID); err != nil {
		return fmt.Errorf("sync like count: %w", err)
	}
	return nil
}

// UpdateStatus 更新评论状态。
func (r *CommentRepo) UpdateStatus(ctx context.Context, commentID, postID string, status int8) error {
	query := fmt.Sprintf(
		"UPDATE %s SET status = ? WHERE comment_id = ?",
		r.router.TableName(postID),
	)
	if _, err := r.db.ExecContext(ctx, query, status, commentID); err != nil {
		return fmt.Errorf("update comment status: %w", err)
	}
	return nil
}

// FindByCommentID 根据 comment_id 查询评论。
func (r *CommentRepo) FindByCommentID(ctx context.Context, commentID, postID string) (*model.Comment, error) {
	query := fmt.Sprintf(`SELECT id, comment_id, post_id, parent_id, author, author_ip, content,
	status, like_count, is_hot, created_at, updated_at
	FROM %s WHERE comment_id = ? LIMIT 1`, r.router.TableName(postID))

	row := r.db.QueryRowContext(ctx, query, commentID)
	comment, err := scanComment(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find comment by id: %w", err)
	}
	return comment, nil
}

// ParseCursor 解析游标 "timestamp_id"。
func ParseCursor(cursor string) (time.Time, int64, error) {
	if strings.TrimSpace(cursor) == "" {
		return time.Time{}, 0, nil
	}

	parts := strings.SplitN(cursor, "_", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, fmt.Errorf("invalid cursor format")
	}

	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("parse cursor timestamp: %w", err)
	}
	cursorID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("parse cursor id: %w", err)
	}

	return time.UnixMilli(timestamp), cursorID, nil
}

// BuildCursor 构建游标 "timestamp_id"。
func BuildCursor(createdAt time.Time, id int64) string {
	return fmt.Sprintf("%d_%d", createdAt.UnixMilli(), id)
}

func scanComment(row *sql.Row) (*model.Comment, error) {
	var (
		comment   model.Comment
		parentID  sql.NullString
		authorIP  sql.NullString
		isHot     int
		updatedAt sql.NullTime
	)

	err := row.Scan(
		&comment.ID,
		&comment.CommentID,
		&comment.PostID,
		&parentID,
		&comment.Author,
		&authorIP,
		&comment.Content,
		&comment.Status,
		&comment.LikeCount,
		&isHot,
		&comment.CreatedAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	comment.ParentID = nullableValue(parentID)
	comment.AuthorIP = nullableValue(authorIP)
	comment.IsHot = isHot == 1
	if updatedAt.Valid {
		comment.UpdatedAt = updatedAt.Time
	}
	return &comment, nil
}

func scanComments(rows *sql.Rows) ([]model.Comment, error) {
	comments := make([]model.Comment, 0, 32)
	for rows.Next() {
		comment, err := scanCommentFromRows(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, *comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	return comments, nil
}

func scanCommentFromRows(rows *sql.Rows) (*model.Comment, error) {
	var (
		comment   model.Comment
		parentID  sql.NullString
		authorIP  sql.NullString
		isHot     int
		updatedAt sql.NullTime
	)

	err := rows.Scan(
		&comment.ID,
		&comment.CommentID,
		&comment.PostID,
		&parentID,
		&comment.Author,
		&authorIP,
		&comment.Content,
		&comment.Status,
		&comment.LikeCount,
		&isHot,
		&comment.CreatedAt,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan comment row: %w", err)
	}

	comment.ParentID = nullableValue(parentID)
	comment.AuthorIP = nullableValue(authorIP)
	comment.IsHot = isHot == 1
	if updatedAt.Valid {
		comment.UpdatedAt = updatedAt.Time
	}
	return &comment, nil
}

func nullableString(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableValue(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func boolToTinyInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
