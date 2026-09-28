package repository

import (
	"context"
	"database/sql"
	"errors"

	"warta/internal/model"
	"warta/internal/pagination"
)

const (
	SortNewest  = "newest"
	SortOldest  = "oldest"
	SortTitle   = "title"
	SortUpdated = "updated"
)

var articleOrder = map[string]string{
	SortNewest:  "COALESCE(a.published_at, a.created_at) DESC, a.id DESC",
	SortOldest:  "COALESCE(a.published_at, a.created_at) ASC, a.id ASC",
	SortTitle:   "a.title ASC, a.id ASC",
	SortUpdated: "a.updated_at DESC, a.id DESC",
}

func ValidSort(sort string) bool {
	_, ok := articleOrder[sort]
	return ok
}

type ArticleFilter struct {
	Query        string
	CategorySlug string
	TagSlug      string
	AuthorID     int64
	// Status kosong berarti semua status.
	Status model.ArticleStatus
	Sort   string
}

type ArticleRepository interface {
	// Create menyimpan article beserta tag-nya dalam satu transaksi. Tag yang
	// belum ada dibuat otomatis.
	Create(ctx context.Context, a *model.Article, tags []string) error
	// Update mengganti tag hanya bila tags tidak nil.
	Update(ctx context.Context, a *model.Article, tags *[]string) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (model.Article, error)
	FindBySlug(ctx context.Context, slug string) (model.Article, error)
	List(ctx context.Context, f ArticleFilter, p pagination.Params) ([]model.Article, int64, error)
	SlugsWithPrefix(ctx context.Context, base string, excludeID int64) ([]string, error)
}

type articleRepository struct {
	db *sql.DB
}

func NewArticleRepository(db *sql.DB) ArticleRepository {
	return &articleRepository{db: db}
}

// excerptSource cukup panjang untuk cuplikan tanpa membaca seluruh isi article
// di halaman daftar.
const (
	excerptSource = 400
	excerptLength = 200
)

const articleFrom = `
FROM articles a
JOIN users u ON u.id = a.author_id
JOIN categories c ON c.id = a.category_id`

func articleSelect(contentColumn string) string {
	return `
SELECT a.id, a.title, a.slug, ` + contentColumn + `, a.status,
       a.author_id, u.name, a.category_id, c.name, c.slug,
       (SELECT COUNT(*) FROM comments cm WHERE cm.article_id = a.id),
       a.published_at, a.created_at, a.updated_at` + articleFrom
}

func scanArticle(row interface{ Scan(...any) error }, a *model.Article, content *string) error {
	var publishedAt sql.NullTime
	err := row.Scan(
		&a.ID, &a.Title, &a.Slug, content, &a.Status,
		&a.AuthorID, &a.AuthorName, &a.CategoryID, &a.CategoryName, &a.CategorySlug,
		&a.CommentCount, &publishedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	a.PublishedAt = nullTimePtr(publishedAt)
	return err
}

func (r *articleRepository) Create(ctx context.Context, a *model.Article, tags []string) error {
	return withTx(ctx, r.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO articles (author_id, category_id, title, slug, content, status, published_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			a.AuthorID, a.CategoryID, a.Title, a.Slug, a.Content, a.Status, a.PublishedAt)
		if err != nil {
			return mapError(err)
		}

		a.ID, err = result.LastInsertId()
		if err != nil {
			return err
		}

		return replaceTags(ctx, tx, a.ID, tags)
	})
}

func (r *articleRepository) Update(ctx context.Context, a *model.Article, tags *[]string) error {
	return withTx(ctx, r.db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE articles
			SET category_id = ?, title = ?, slug = ?, content = ?, status = ?, published_at = ?
			WHERE id = ?`,
			a.CategoryID, a.Title, a.Slug, a.Content, a.Status, a.PublishedAt, a.ID)
		if err != nil {
			return mapError(err)
		}

		if tags == nil {
			return nil
		}
		return replaceTags(ctx, tx, a.ID, *tags)
	})
}

func replaceTags(ctx context.Context, tx *sql.Tx, articleID int64, names []string) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM article_tags WHERE article_id = ?", articleID); err != nil {
		return err
	}

	ids, err := ensureTags(ctx, tx, names)
	if err != nil || len(ids) == 0 {
		return err
	}

	args := make([]any, 0, len(ids)*2)
	for _, id := range ids {
		args = append(args, articleID, id)
	}

	values := "(?, ?)"
	for i := 1; i < len(ids); i++ {
		values += ", (?, ?)"
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO article_tags (article_id, tag_id) VALUES "+values, args...)
	return mapError(err)
}

func (r *articleRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM articles WHERE id = ?", id)
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

func (r *articleRepository) FindByID(ctx context.Context, id int64) (model.Article, error) {
	return r.findOne(ctx, "a.id = ?", id)
}

func (r *articleRepository) FindBySlug(ctx context.Context, slug string) (model.Article, error) {
	return r.findOne(ctx, "a.slug = ?", slug)
}

func (r *articleRepository) findOne(ctx context.Context, condition string, arg any) (model.Article, error) {
	var a model.Article
	err := scanArticle(r.db.QueryRowContext(ctx, articleSelect("a.content")+" WHERE "+condition, arg), &a, &a.Content)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Article{}, ErrNotFound
	}
	if err != nil {
		return model.Article{}, err
	}

	a.Excerpt = model.Excerpt(a.Content, excerptLength)

	articles := []*model.Article{&a}
	if err := r.loadTags(ctx, articles); err != nil {
		return model.Article{}, err
	}
	return a, nil
}

func (r *articleRepository) List(ctx context.Context, f ArticleFilter, p pagination.Params) ([]model.Article, int64, error) {
	var conditions []string
	var args []any

	if f.Status != "" {
		conditions = append(conditions, "a.status = ?")
		args = append(args, f.Status)
	}
	if f.AuthorID > 0 {
		conditions = append(conditions, "a.author_id = ?")
		args = append(args, f.AuthorID)
	}
	if f.CategorySlug != "" {
		conditions = append(conditions, "c.slug = ?")
		args = append(args, f.CategorySlug)
	}
	if f.TagSlug != "" {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM article_tags at JOIN tags t ON t.id = at.tag_id
			WHERE at.article_id = a.id AND t.slug = ?)`)
		args = append(args, f.TagSlug)
	}
	if f.Query != "" {
		pattern := "%" + escapeLike(f.Query) + "%"
		conditions = append(conditions, "(a.title LIKE ? OR a.content LIKE ?)")
		args = append(args, pattern, pattern)
	}
	where := whereClause(conditions)

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*)"+articleFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	order, ok := articleOrder[f.Sort]
	if !ok {
		order = articleOrder[SortNewest]
	}

	rows, err := r.db.QueryContext(ctx,
		articleSelect("LEFT(a.content, ?)")+where+" ORDER BY "+order+" LIMIT ? OFFSET ?",
		append(append([]any{excerptSource}, args...), p.Limit(), p.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var articles []model.Article
	for rows.Next() {
		var a model.Article
		var head string
		if err := scanArticle(rows, &a, &head); err != nil {
			return nil, 0, err
		}
		a.Excerpt = model.Excerpt(head, excerptLength)
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	pointers := make([]*model.Article, len(articles))
	for i := range articles {
		pointers[i] = &articles[i]
	}
	if err := r.loadTags(ctx, pointers); err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// loadTags mengisi tag semua article dengan satu query.
func (r *articleRepository) loadTags(ctx context.Context, articles []*model.Article) error {
	if len(articles) == 0 {
		return nil
	}

	byID := make(map[int64]*model.Article, len(articles))
	args := make([]any, 0, len(articles))
	for _, a := range articles {
		a.Tags = []model.Tag{}
		byID[a.ID] = a
		args = append(args, a.ID)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT at.article_id, t.id, t.name, t.slug, t.created_at
		FROM article_tags at JOIN tags t ON t.id = at.tag_id
		WHERE at.article_id IN (`+placeholders(len(args))+`)
		ORDER BY t.name ASC`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var articleID int64
		var t model.Tag
		if err := rows.Scan(&articleID, &t.ID, &t.Name, &t.Slug, &t.CreatedAt); err != nil {
			return err
		}
		if a := byID[articleID]; a != nil {
			a.Tags = append(a.Tags, t)
		}
	}
	return rows.Err()
}

func (r *articleRepository) SlugsWithPrefix(ctx context.Context, base string, excludeID int64) ([]string, error) {
	return slugsWithPrefix(ctx, r.db, "articles", base, excludeID)
}
