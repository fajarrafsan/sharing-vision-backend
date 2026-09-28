package repository

import (
	"context"
	"database/sql"
	"errors"

	"warta/internal/model"
)

type CategoryRepository interface {
	Create(ctx context.Context, c *model.Category) error
	Update(ctx context.Context, c *model.Category) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (model.Category, error)
	FindBySlug(ctx context.Context, slug string) (model.Category, error)
	List(ctx context.Context) ([]model.Category, error)
	NameTaken(ctx context.Context, name string, excludeID int64) (bool, error)
	SlugsWithPrefix(ctx context.Context, base string, excludeID int64) ([]string, error)
}

type categoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

const categorySelect = `
SELECT c.id, c.name, c.slug, c.description, c.created_at, c.updated_at,
       (SELECT COUNT(*) FROM articles a WHERE a.category_id = c.id AND a.status = 'published')
FROM categories c`

func scanCategory(row interface{ Scan(...any) error }, c *model.Category) error {
	return row.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.CreatedAt, &c.UpdatedAt, &c.ArticleCount)
}

func (r *categoryRepository) Create(ctx context.Context, c *model.Category) error {
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO categories (name, slug, description) VALUES (?, ?, ?)",
		c.Name, c.Slug, c.Description)
	if err != nil {
		return mapError(err)
	}

	c.ID, err = result.LastInsertId()
	return err
}

func (r *categoryRepository) Update(ctx context.Context, c *model.Category) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE categories SET name = ?, slug = ?, description = ? WHERE id = ?",
		c.Name, c.Slug, c.Description, c.ID)
	return mapError(err)
}

func (r *categoryRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM categories WHERE id = ?", id)
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

func (r *categoryRepository) FindByID(ctx context.Context, id int64) (model.Category, error) {
	return r.findOne(ctx, "c.id = ?", id)
}

func (r *categoryRepository) FindBySlug(ctx context.Context, slug string) (model.Category, error) {
	return r.findOne(ctx, "c.slug = ?", slug)
}

func (r *categoryRepository) findOne(ctx context.Context, condition string, arg any) (model.Category, error) {
	var c model.Category
	err := scanCategory(r.db.QueryRowContext(ctx, categorySelect+" WHERE "+condition, arg), &c)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Category{}, ErrNotFound
	}
	return c, err
}

func (r *categoryRepository) List(ctx context.Context) ([]model.Category, error) {
	rows, err := r.db.QueryContext(ctx, categorySelect+" ORDER BY c.name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var c model.Category
		if err := scanCategory(rows, &c); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *categoryRepository) NameTaken(ctx context.Context, name string, excludeID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM categories WHERE name = ? AND id <> ?)", name, excludeID,
	).Scan(&exists)
	return exists, err
}

func (r *categoryRepository) SlugsWithPrefix(ctx context.Context, base string, excludeID int64) ([]string, error) {
	return slugsWithPrefix(ctx, r.db, "categories", base, excludeID)
}
