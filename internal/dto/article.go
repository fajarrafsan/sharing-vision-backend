package dto

import (
	"strings"
	"time"

	"sharing-vision-backend/internal/model"
)

type ArticleRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Category string `json:"category"`
	Status   string `json:"status"`
}

func (r *ArticleRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Content = strings.TrimSpace(r.Content)
	r.Category = strings.TrimSpace(r.Category)
	r.Status = strings.ToLower(strings.TrimSpace(r.Status))
}

func (r ArticleRequest) ToModel() model.Article {
	return model.Article{
		Title:    r.Title,
		Content:  r.Content,
		Category: r.Category,
		Status:   r.Status,
	}
}

type ArticleResponse struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	CreatedDate time.Time `json:"created_date"`
	UpdatedDate time.Time `json:"updated_date"`
}

func NewArticleResponse(a model.Article) ArticleResponse {
	return ArticleResponse{
		ID:          a.ID,
		Title:       a.Title,
		Content:     a.Content,
		Category:    a.Category,
		Status:      a.Status,
		CreatedDate: a.CreatedDate,
		UpdatedDate: a.UpdatedDate,
	}
}

func NewArticleResponses(articles []model.Article) []ArticleResponse {
	responses := []ArticleResponse{}
	for _, a := range articles {
		responses = append(responses, NewArticleResponse(a))
	}
	return responses
}
