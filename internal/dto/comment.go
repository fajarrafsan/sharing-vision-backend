package dto

import (
	"strings"
	"time"

	"warta/internal/model"
)

type CommentRequest struct {
	Body string `json:"body"`
}

func (r *CommentRequest) Normalize() {
	r.Body = strings.TrimSpace(r.Body)
}

type CommentResponse struct {
	ID        int64          `json:"id"`
	ArticleID int64          `json:"article_id"`
	Body      string         `json:"body"`
	Author    AuthorResponse `json:"author"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func NewCommentResponse(c model.Comment) CommentResponse {
	return CommentResponse{
		ID:        c.ID,
		ArticleID: c.ArticleID,
		Body:      c.Body,
		Author:    AuthorResponse{ID: c.UserID, Name: c.AuthorName},
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func NewCommentResponses(comments []model.Comment) []CommentResponse {
	responses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		responses = append(responses, NewCommentResponse(c))
	}
	return responses
}
