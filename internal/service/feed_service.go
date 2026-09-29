package service

import (
	"context"

	"warta/internal/apperr"
	"warta/internal/model"
	"warta/internal/pagination"
	"warta/internal/repository"
)

// FeedService menyediakan data untuk sitemap dan RSS: hanya artikel terbit.
type FeedService interface {
	// Latest adalah artikel terbit terbaru, untuk RSS.
	Latest(ctx context.Context, limit int) ([]model.Article, error)
	// Sitemap adalah artikel terbit yang terakhir diubah dan kategori yang
	// punya artikel.
	Sitemap(ctx context.Context, limit int) ([]model.Article, []model.Category, error)
}

type feedService struct {
	articles   repository.ArticleRepository
	categories repository.CategoryRepository
}

func NewFeedService(articles repository.ArticleRepository, categories repository.CategoryRepository) FeedService {
	return &feedService{articles: articles, categories: categories}
}

func (s *feedService) Latest(ctx context.Context, limit int) ([]model.Article, error) {
	return s.published(ctx, repository.SortNewest, limit)
}

func (s *feedService) Sitemap(ctx context.Context, limit int) ([]model.Article, []model.Category, error) {
	articles, err := s.published(ctx, repository.SortUpdated, limit)
	if err != nil {
		return nil, nil, err
	}

	all, err := s.categories.List(ctx)
	if err != nil {
		return nil, nil, apperr.Internal(err)
	}
	categories := make([]model.Category, 0, len(all))
	for _, c := range all {
		if c.ArticleCount > 0 {
			categories = append(categories, c)
		}
	}
	return articles, categories, nil
}

func (s *feedService) published(ctx context.Context, sort string, limit int) ([]model.Article, error) {
	articles, _, err := s.articles.List(ctx,
		repository.ArticleFilter{Status: model.StatusPublished, Sort: sort},
		pagination.Params{Page: 1, PerPage: limit})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return articles, nil
}
