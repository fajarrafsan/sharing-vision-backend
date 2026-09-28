package repository

import (
	"context"
	"database/sql"
	"errors"

	"warta/internal/model"
	"warta/internal/pagination"
	"warta/internal/slug"
)

type TagRepository interface {
	Create(ctx context.Context, t *model.Tag) error
	Update(ctx context.Context, t *model.Tag) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (model.Tag, error)
	List(ctx context.Context, query string, p pagination.Params) ([]model.Tag, int64, error)
	NameTaken(ctx context.Context, name string, excludeID int64) (bool, error)
	SlugsWithPrefix(ctx context.Context, base string, excludeID int64) ([]string, error)
}

type tagRepository struct {
	db *sql.DB
}

func NewTagRepository(db *sql.DB) TagRepository {
	return &tagRepository{db: db}
}

const tagSelect = `
SELECT t.id, t.name, t.slug, t.created_at,
       (SELECT COUNT(*) FROM article_tags at JOIN articles a ON a.id = at.article_id
        WHERE at.tag_id = t.id AND a.status = 'published') AS article_count
FROM tags t`

func scanTag(row interface{ Scan(...any) error }, t *model.Tag) error {
	return row.Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt, &t.ArticleCount)
}

func (r *tagRepository) Create(ctx context.Context, t *model.Tag) error {
	id, err := insertTag(ctx, r.db, t.Name, t.Slug)
	if err != nil {
		return err
	}
	t.ID = id
	return nil
}

func (r *tagRepository) Update(ctx context.Context, t *model.Tag) error {
	_, err := r.db.ExecContext(ctx, "UPDATE tags SET name = ?, slug = ? WHERE id = ?", t.Name, t.Slug, t.ID)
	return mapError(err)
}

func (r *tagRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM tags WHERE id = ?", id)
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

func (r *tagRepository) FindByID(ctx context.Context, id int64) (model.Tag, error) {
	var t model.Tag
	err := scanTag(r.db.QueryRowContext(ctx, tagSelect+" WHERE t.id = ?", id), &t)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Tag{}, ErrNotFound
	}
	return t, err
}

func (r *tagRepository) List(ctx context.Context, query string, p pagination.Params) ([]model.Tag, int64, error) {
	where := ""
	var args []any
	if query != "" {
		where = " WHERE t.name LIKE ?"
		args = append(args, "%"+escapeLike(query)+"%")
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tags t"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		tagSelect+where+" ORDER BY article_count DESC, t.name ASC LIMIT ? OFFSET ?",
		append(args, p.Limit(), p.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := scanTag(rows, &t); err != nil {
			return nil, 0, err
		}
		tags = append(tags, t)
	}
	return tags, total, rows.Err()
}

func (r *tagRepository) NameTaken(ctx context.Context, name string, excludeID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM tags WHERE name = ? AND id <> ?)", name, excludeID,
	).Scan(&exists)
	return exists, err
}

func (r *tagRepository) SlugsWithPrefix(ctx context.Context, base string, excludeID int64) ([]string, error) {
	return slugsWithPrefix(ctx, r.db, "tags", base, excludeID)
}

func insertTag(ctx context.Context, q DBTX, name, tagSlug string) (int64, error) {
	result, err := q.ExecContext(ctx, "INSERT INTO tags (name, slug) VALUES (?, ?)", name, tagSlug)
	if err != nil {
		return 0, mapError(err)
	}
	return result.LastInsertId()
}

// ensureTags mengembalikan id untuk setiap nama tag, membuat tag yang belum
// ada. Nama dicari satu per satu supaya perbandingannya mengikuti collation
// kolom (tidak peka huruf besar dan aksen), sama seperti indeks uniknya.
func ensureTags(ctx context.Context, tx *sql.Tx, names []string) ([]int64, error) {
	ids := make([]int64, 0, len(names))
	seen := make(map[int64]bool, len(names))

	for _, name := range names {
		var id int64
		err := tx.QueryRowContext(ctx, "SELECT id FROM tags WHERE name = ?", name).Scan(&id)
		if err == nil {
			// Dua nama yang berbeda bisa jatuh ke tag yang sama, misalnya
			// "cafe" dan "café".
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		base := slug.Make(name, "tag", 60)
		taken, err := slugsWithPrefix(ctx, tx, "tags", base, 0)
		if err != nil {
			return nil, err
		}

		id, err = insertTag(ctx, tx, name, slug.Unique(base, taken))
		if err != nil {
			return nil, err
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}
