package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"warta/internal/model"
	"warta/internal/pagination"
)

type CommentRepository interface {
	Create(ctx context.Context, c *model.Comment) error
	UpdateBody(ctx context.Context, id int64, body string) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (model.Comment, error)
	// ListByArticle hanya memuat komentar yang tidak disembunyikan.
	ListByArticle(ctx context.Context, articleID int64, p pagination.Params) ([]model.Comment, int64, error)

	// HasRecentDuplicate memeriksa apakah user baru saja mengirim komentar
	// dengan isi yang sama di artikel yang sama.
	HasRecentDuplicate(ctx context.Context, articleID, userID int64, body string, since time.Time) (bool, error)
	// Report mencatat laporan (satu per user per komentar) dan mengembalikan
	// jumlah laporan komentar itu.
	Report(ctx context.Context, commentID, userID int64, reason string) (int, error)
	SetHidden(ctx context.Context, id int64, hidden bool) error
	ClearReports(ctx context.Context, id int64) error
	// ListReported memuat komentar yang dilaporkan atau disembunyikan.
	ListReported(ctx context.Context, p pagination.Params) ([]model.ReportedComment, int64, error)
}

type commentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) CommentRepository {
	return &commentRepository{db: db}
}

const commentSelect = `
SELECT cm.id, cm.article_id, cm.user_id, u.name, cm.body, cm.hidden_at, cm.created_at, cm.updated_at
FROM comments cm
JOIN users u ON u.id = cm.user_id`

func scanComment(row interface{ Scan(...any) error }, c *model.Comment, extra ...any) error {
	var hiddenAt sql.NullTime
	dest := append([]any{&c.ID, &c.ArticleID, &c.UserID, &c.AuthorName, &c.Body, &hiddenAt, &c.CreatedAt, &c.UpdatedAt}, extra...)
	err := row.Scan(dest...)
	c.HiddenAt = nullTimePtr(hiddenAt)
	return err
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
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM comments WHERE article_id = ? AND hidden_at IS NULL", articleID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		commentSelect+" WHERE cm.article_id = ? AND cm.hidden_at IS NULL ORDER BY cm.created_at ASC, cm.id ASC LIMIT ? OFFSET ?",
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

func (r *commentRepository) HasRecentDuplicate(ctx context.Context, articleID, userID int64, body string, since time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM comments
			WHERE article_id = ? AND user_id = ? AND body = ? AND created_at >= ?)`,
		articleID, userID, body, since,
	).Scan(&exists)
	return exists, err
}

func (r *commentRepository) Report(ctx context.Context, commentID, userID int64, reason string) (int, error) {
	_, err := r.db.ExecContext(ctx,
		"INSERT IGNORE INTO comment_reports (comment_id, user_id, reason) VALUES (?, ?, ?)",
		commentID, userID, reason)
	if err != nil {
		return 0, mapError(err)
	}

	var count int
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM comment_reports WHERE comment_id = ?", commentID).Scan(&count)
	return count, err
}

// SetHidden tidak mengubah updated_at, karena updated_at menandakan isi
// komentar yang diubah penulisnya.
func (r *commentRepository) SetHidden(ctx context.Context, id int64, hidden bool) error {
	query := "UPDATE comments SET hidden_at = NULL, updated_at = updated_at WHERE id = ?"
	if hidden {
		query = "UPDATE comments SET hidden_at = COALESCE(hidden_at, CURRENT_TIMESTAMP), updated_at = updated_at WHERE id = ?"
	}
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *commentRepository) ClearReports(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM comment_reports WHERE comment_id = ?", id)
	return err
}

func (r *commentRepository) ListReported(ctx context.Context, p pagination.Params) ([]model.ReportedComment, int64, error) {
	const where = ` WHERE cm.hidden_at IS NOT NULL OR EXISTS (SELECT 1 FROM comment_reports cr WHERE cr.comment_id = cm.id)`

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM comments cm"+where).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT cm.id, cm.article_id, cm.user_id, u.name, cm.body, cm.hidden_at, cm.created_at, cm.updated_at,
		       a.title, a.slug,
		       COUNT(cr.user_id),
		       COALESCE(SUM(cr.reason = 'spam'), 0),
		       COALESCE(SUM(cr.reason = 'abusive'), 0),
		       COALESCE(SUM(cr.reason = 'other'), 0),
		       MAX(cr.created_at)
		FROM comments cm
		JOIN users u ON u.id = cm.user_id
		JOIN articles a ON a.id = cm.article_id
		LEFT JOIN comment_reports cr ON cr.comment_id = cm.id`+where+`
		GROUP BY cm.id, u.name, a.title, a.slug
		ORDER BY COUNT(cr.user_id) DESC, MAX(cr.created_at) DESC, cm.id DESC
		LIMIT ? OFFSET ?`,
		p.Limit(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reported []model.ReportedComment
	for rows.Next() {
		var c model.ReportedComment
		var spam, abusive, other int
		var last sql.NullTime
		err := scanComment(rows, &c.Comment, &c.ArticleTitle, &c.ArticleSlug, &c.Reports, &spam, &abusive, &other, &last)
		if err != nil {
			return nil, 0, err
		}
		c.ReasonCounts = map[string]int{model.ReportSpam: spam, model.ReportAbusive: abusive, model.ReportOther: other}
		c.LastReportedAt = nullTimePtr(last)
		reported = append(reported, c)
	}
	return reported, total, rows.Err()
}
