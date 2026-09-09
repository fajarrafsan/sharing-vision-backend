package service

import (
	"errors"

	"sharing-vision-backend/internal/apperr"
	"sharing-vision-backend/internal/dto"
	"sharing-vision-backend/internal/repository"
	"sharing-vision-backend/internal/validation"
)

type ArticleService interface {
	Create(req dto.ArticleRequest) (dto.ArticleResponse, error)
	List(limit, offset int) ([]dto.ArticleResponse, error)
	GetByID(id int64) (dto.ArticleResponse, error)
	Update(id int64, req dto.ArticleRequest) (dto.ArticleResponse, error)
	Delete(id int64) error
}

type articleService struct {
	repo         repository.ArticleRepository
	defaultLimit int
	maxLimit     int
}

func NewArticleService(repo repository.ArticleRepository, defaultLimit, maxLimit int) ArticleService {
	return &articleService{
		repo:         repo,
		defaultLimit: defaultLimit,
		maxLimit:     maxLimit,
	}
}

func (s *articleService) Create(req dto.ArticleRequest) (dto.ArticleResponse, error) {
	req.Normalize()

	if problems := validation.ValidateArticle(req); len(problems) > 0 {
		return dto.ArticleResponse{}, apperr.Validation(problems)
	}

	article := req.ToModel()
	if err := s.repo.Create(&article); err != nil {
		return dto.ArticleResponse{}, apperr.Internal(err)
	}

	return dto.NewArticleResponse(article), nil
}

func (s *articleService) List(limit, offset int) ([]dto.ArticleResponse, error) {
	limit, offset = s.clampPaging(limit, offset)

	articles, err := s.repo.FindAll(limit, offset)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	return dto.NewArticleResponses(articles), nil
}

func (s *articleService) GetByID(id int64) (dto.ArticleResponse, error) {
	article, err := s.repo.FindByID(id)
	if err != nil {
		return dto.ArticleResponse{}, translate(err)
	}

	return dto.NewArticleResponse(article), nil
}

func (s *articleService) Update(id int64, req dto.ArticleRequest) (dto.ArticleResponse, error) {
	req.Normalize()

	if problems := validation.ValidateArticle(req); len(problems) > 0 {
		return dto.ArticleResponse{}, apperr.Validation(problems)
	}

	article, err := s.repo.FindByID(id)
	if err != nil {
		return dto.ArticleResponse{}, translate(err)
	}

	article.Title = req.Title
	article.Content = req.Content
	article.Category = req.Category
	article.Status = req.Status

	if err := s.repo.Update(&article); err != nil {
		return dto.ArticleResponse{}, apperr.Internal(err)
	}

	return dto.NewArticleResponse(article), nil
}

func (s *articleService) Delete(id int64) error {
	if err := s.repo.Delete(id); err != nil {
		return translate(err)
	}
	return nil
}

func (s *articleService) clampPaging(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = s.defaultLimit
	}
	if limit > s.maxLimit {
		limit = s.maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func translate(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return apperr.NotFound("article tidak ditemukan")
	}
	return apperr.Internal(err)
}
