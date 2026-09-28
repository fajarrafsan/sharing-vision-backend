package service

import (
	"context"
	"errors"

	"warta/internal/apperr"
	"warta/internal/auth"
	"warta/internal/dto"
	"warta/internal/model"
	"warta/internal/pagination"
	"warta/internal/repository"
	"warta/internal/validation"
)

type CommentService interface {
	List(ctx context.Context, actor auth.Actor, articleID int64, p pagination.Params) ([]dto.CommentResponse, pagination.Meta, error)
	Create(ctx context.Context, actor auth.Actor, articleID int64, req dto.CommentRequest) (dto.CommentResponse, error)
	Update(ctx context.Context, actor auth.Actor, id int64, req dto.CommentRequest) (dto.CommentResponse, error)
	Delete(ctx context.Context, actor auth.Actor, id int64) error
}

type commentService struct {
	comments repository.CommentRepository
	articles repository.ArticleRepository
}

func NewCommentService(comments repository.CommentRepository, articles repository.ArticleRepository) CommentService {
	return &commentService{comments: comments, articles: articles}
}

var errCommentNotFound = apperr.NotFound("komentar tidak ditemukan")

func (s *commentService) List(ctx context.Context, actor auth.Actor, articleID int64, p pagination.Params) ([]dto.CommentResponse, pagination.Meta, error) {
	if _, err := s.visibleArticle(ctx, actor, articleID); err != nil {
		return nil, pagination.Meta{}, err
	}

	comments, total, err := s.comments.ListByArticle(ctx, articleID, p)
	if err != nil {
		return nil, pagination.Meta{}, apperr.Internal(err)
	}
	return dto.NewCommentResponses(comments), pagination.NewMeta(p, total), nil
}

func (s *commentService) Create(ctx context.Context, actor auth.Actor, articleID int64, req dto.CommentRequest) (dto.CommentResponse, error) {
	article, err := s.visibleArticle(ctx, actor, articleID)
	if err != nil {
		return dto.CommentResponse{}, err
	}
	if !article.IsPublished() {
		return dto.CommentResponse{}, apperr.Forbidden("komentar hanya bisa ditambahkan ke article yang sudah terbit")
	}

	req.Normalize()
	if problems := validation.ValidateComment(req); len(problems) > 0 {
		return dto.CommentResponse{}, apperr.Validation(problems)
	}

	comment := model.Comment{ArticleID: articleID, UserID: actor.ID, Body: req.Body}
	if err := s.comments.Create(ctx, &comment); err != nil {
		if errors.Is(err, repository.ErrMissingReference) {
			return dto.CommentResponse{}, errArticleNotFound
		}
		return dto.CommentResponse{}, apperr.Internal(err)
	}

	return dto.NewCommentResponse(comment), nil
}

// Update hanya untuk pemilik komentar. Admin boleh menghapus komentar orang
// lain, tapi tidak mengubah kata-katanya.
func (s *commentService) Update(ctx context.Context, actor auth.Actor, id int64, req dto.CommentRequest) (dto.CommentResponse, error) {
	comment, err := s.find(ctx, id)
	if err != nil {
		return dto.CommentResponse{}, err
	}
	if comment.UserID != actor.ID {
		return dto.CommentResponse{}, apperr.Forbidden("hanya penulis komentar yang bisa mengubahnya")
	}

	req.Normalize()
	if problems := validation.ValidateComment(req); len(problems) > 0 {
		return dto.CommentResponse{}, apperr.Validation(problems)
	}

	if err := s.comments.UpdateBody(ctx, id, req.Body); err != nil {
		return dto.CommentResponse{}, apperr.Internal(err)
	}

	updated, err := s.find(ctx, id)
	if err != nil {
		return dto.CommentResponse{}, err
	}
	return dto.NewCommentResponse(updated), nil
}

func (s *commentService) Delete(ctx context.Context, actor auth.Actor, id int64) error {
	comment, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	if comment.UserID != actor.ID && !actor.IsAdmin() {
		return apperr.Forbidden("hanya penulis komentar dan admin yang bisa menghapusnya")
	}

	err = s.comments.Delete(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return errCommentNotFound
	}
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (s *commentService) find(ctx context.Context, id int64) (model.Comment, error) {
	comment, err := s.comments.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Comment{}, errCommentNotFound
	}
	if err != nil {
		return model.Comment{}, apperr.Internal(err)
	}
	return comment, nil
}

func (s *commentService) visibleArticle(ctx context.Context, actor auth.Actor, id int64) (model.Article, error) {
	article, err := s.articles.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) || err == nil && !canView(actor, article) {
		return model.Article{}, errArticleNotFound
	}
	if err != nil {
		return model.Article{}, apperr.Internal(err)
	}
	return article, nil
}
