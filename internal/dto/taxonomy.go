package dto

import (
	"strings"
	"time"

	"warta/internal/model"
)

type CategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r *CategoryRequest) Normalize() {
	r.Name = collapseSpaces(r.Name)
	r.Description = strings.TrimSpace(r.Description)
}

type CategoryResponse struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description"`
	ArticleCount int       `json:"article_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewCategoryResponse(c model.Category) CategoryResponse {
	return CategoryResponse{
		ID:           c.ID,
		Name:         c.Name,
		Slug:         c.Slug,
		Description:  c.Description,
		ArticleCount: c.ArticleCount,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

func NewCategoryResponses(categories []model.Category) []CategoryResponse {
	responses := make([]CategoryResponse, 0, len(categories))
	for _, c := range categories {
		responses = append(responses, NewCategoryResponse(c))
	}
	return responses
}

// CategoryRef adalah bentuk ringkas kategori di dalam article.
type CategoryRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type TagRequest struct {
	Name string `json:"name"`
}

func (r *TagRequest) Normalize() {
	r.Name = NormalizeTag(r.Name)
}

// NormalizeTag membuat tag huruf kecil dengan spasi tunggal, supaya "Go Lang"
// dan " go  lang" dianggap tag yang sama.
func NormalizeTag(name string) string {
	return strings.ToLower(collapseSpaces(name))
}

type TagResponse struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	ArticleCount int       `json:"article_count"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewTagResponse(t model.Tag) TagResponse {
	return TagResponse{
		ID:           t.ID,
		Name:         t.Name,
		Slug:         t.Slug,
		ArticleCount: t.ArticleCount,
		CreatedAt:    t.CreatedAt,
	}
}

func NewTagResponses(tags []model.Tag) []TagResponse {
	responses := make([]TagResponse, 0, len(tags))
	for _, t := range tags {
		responses = append(responses, NewTagResponse(t))
	}
	return responses
}

// TagRef adalah bentuk ringkas tag di dalam article.
type TagRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
