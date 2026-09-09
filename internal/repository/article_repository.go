package repository

import (
	"database/sql"
	"errors"
	"time"

	"sharing-vision-backend/internal/model"
)

var ErrNotFound = errors.New("article tidak ditemukan")

type ArticleRepository interface {
	Create(a *model.Article) error
	FindAll(limit, offset int) ([]model.Article, error)
	FindByID(id int64) (model.Article, error)
	Update(a *model.Article) error
	Delete(id int64) error
}

type articleRepository struct {
	db *sql.DB
}

func NewArticleRepository(db *sql.DB) ArticleRepository {
	return &articleRepository{db: db}
}

const columns = "id, title, content, category, created_date, updated_date, status"

func (r *articleRepository) Create(a *model.Article) error {
	now := time.Now()

	query := "INSERT INTO posts (title, content, category, created_date, updated_date, status) VALUES (?, ?, ?, ?, ?, ?)"
	result, err := r.db.Exec(query, a.Title, a.Content, a.Category, now, now, a.Status)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	a.ID = id
	a.CreatedDate = now
	a.UpdatedDate = now
	return nil
}

func (r *articleRepository) FindAll(limit, offset int) ([]model.Article, error) {
	query := "SELECT " + columns + " FROM posts ORDER BY id DESC LIMIT ? OFFSET ?"

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []model.Article
	for rows.Next() {
		var a model.Article
		err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.CreatedDate, &a.UpdatedDate, &a.Status)
		if err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}

	return articles, rows.Err()
}

func (r *articleRepository) FindByID(id int64) (model.Article, error) {
	query := "SELECT " + columns + " FROM posts WHERE id = ?"

	var a model.Article
	err := r.db.QueryRow(query, id).Scan(
		&a.ID, &a.Title, &a.Content, &a.Category, &a.CreatedDate, &a.UpdatedDate, &a.Status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Article{}, ErrNotFound
	}
	if err != nil {
		return model.Article{}, err
	}

	return a, nil
}

func (r *articleRepository) Update(a *model.Article) error {
	now := time.Now()

	query := "UPDATE posts SET title = ?, content = ?, category = ?, status = ?, updated_date = ? WHERE id = ?"
	_, err := r.db.Exec(query, a.Title, a.Content, a.Category, a.Status, now, a.ID)
	if err != nil {
		return err
	}

	a.UpdatedDate = now
	return nil
}

func (r *articleRepository) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM posts WHERE id = ?", id)
	if err != nil {
		return err
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
