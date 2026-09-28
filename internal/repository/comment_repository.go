package repository

import (
	"context"
	"database/sql"
	"errors"

	"warta/internal/model"
	"warta/internal/pagination"
)

type CommentRepository interface {
	Create(ctx context.Context, c *model.Comment) error
	UpdateBody(ctx context.Context, id int64, body string) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (model.Comment, error)
	ListByArticle(ctx context.Context, articleID int64, p pagination.Params) ([]model.Comment, int64, error)
}

type commentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) CommentRepository {
	return &commentRepository{db: db}
}

const commentSelect = `
SELECT cm.id, cm.article_id, cm.user_id, u.name, cm.body, cm.created_at, cm.updated_at
FROM comments cm
JOIN users u ON u.id = cm.user_id`

func scanComment(row interface{ Scan(...any) error }, c *model.Comment) error {
	return row.Scan(&c.ID, &c.ArticleID, &c.UserID, &c.AuthorName, &c.Body, &c.CreatedAt, &c.UpdatedAt)
}

func (r *commentRepository) Create(ctx context.Context, c *model.Comment) error {
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO comments (article_id, user_id, body) VALUES (?, ?, ?)",
		c.ArticleID, c.UserID, c.Body)
	if err != nil {
		return mapError(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	created, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	*c = created
	return nil
}

func (r *commentRepository) UpdateBody(ctx context.Context, id int64, body string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE comments SET body = ? WHERE id = ?", body, id)
	return mapError(err)
}

func (r *commentRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM comments WHERE id = ?", id)
	if err != nil {
		return mapError(err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *commentRepository) FindByID(ctx context.Context, id int64) (model.Comment, error) {
	var c model.Comment
	err := scanComment(r.db.QueryRowContext(ctx, commentSelect+" WHERE cm.id = ?", id), &c)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Comment{}, ErrNotFound
	}
	return c, err
}

func (r *commentRepository) ListByArticle(ctx context.Context, articleID int64, p pagination.Params) ([]model.Comment, int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM comments WHERE article_id = ?", articleID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		commentSelect+" WHERE cm.article_id = ? ORDER BY cm.created_at ASC, cm.id ASC LIMIT ? OFFSET ?",
		articleID, p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []model.Comment
	for rows.Next() {
		var c model.Comment
		if err := scanComment(rows, &c); err != nil {
			return nil, 0, err
		}
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}
