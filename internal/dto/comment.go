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

type ReportRequest struct {
	Reason string `json:"reason"`
}

func (r *ReportRequest) Normalize() {
	r.Reason = strings.ToLower(strings.TrimSpace(r.Reason))
}

type ReportResponse struct {
	Reported bool `json:"reported"`
	// Hidden menandakan komentar kini disembunyikan karena laporannya
	// mencapai batas.
	Hidden bool `json:"hidden"`
}

type ModerationRequest struct {
	Action string `json:"action"`
}

func (r *ModerationRequest) Normalize() {
	r.Action = strings.ToLower(strings.TrimSpace(r.Action))
}

type ArticleRef struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type ReportedCommentResponse struct {
	CommentResponse
	Article        ArticleRef     `json:"article"`
	Hidden         bool           `json:"hidden"`
	Reports        int            `json:"reports"`
	Reasons        map[string]int `json:"reasons"`
	LastReportedAt *time.Time     `json:"last_reported_at"`
}

func NewReportedCommentResponses(comments []model.ReportedComment) []ReportedCommentResponse {
	responses := make([]ReportedCommentResponse, 0, len(comments))
	for _, c := range comments {
		responses = append(responses, ReportedCommentResponse{
			CommentResponse: NewCommentResponse(c.Comment),
			Article:         ArticleRef{ID: c.ArticleID, Title: c.ArticleTitle, Slug: c.ArticleSlug},
			Hidden:          c.HiddenAt != nil,
			Reports:         c.Reports,
			Reasons:         c.ReasonCounts,
			LastReportedAt:  c.LastReportedAt,
		})
	}
	return responses
}
