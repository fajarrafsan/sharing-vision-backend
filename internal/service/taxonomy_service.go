package service

import (
	"context"
	"errors"
	"strconv"

	"warta/internal/apperr"
	"warta/internal/dto"
	"warta/internal/model"
	"warta/internal/pagination"
	"warta/internal/repository"
	"warta/internal/slug"
	"warta/internal/validation"
)

type CategoryService interface {
	List(ctx context.Context) ([]dto.CategoryResponse, error)
	// Get menerima id maupun slug.
	Get(ctx context.Context, ref string) (dto.CategoryResponse, error)
	Create(ctx context.Context, req dto.CategoryRequest) (dto.CategoryResponse, error)
	Update(ctx context.Context, id int64, req dto.CategoryRequest) (dto.CategoryResponse, error)
	Delete(ctx context.Context, id int64) error
}

type categoryService struct {
	categories repository.CategoryRepository
}

func NewCategoryService(categories repository.CategoryRepository) CategoryService {
	return &categoryService{categories: categories}
}

func (s *categoryService) List(ctx context.Context) ([]dto.CategoryResponse, error) {
	categories, err := s.categories.List(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return dto.NewCategoryResponses(categories), nil
}

func (s *categoryService) Get(ctx context.Context, ref string) (dto.CategoryResponse, error) {
	var category model.Category
	var err error

	if id, ok := parseID(ref); ok {
		category, err = s.categories.FindByID(ctx, id)
	} else {
		category, err = s.categories.FindBySlug(ctx, ref)
	}
	if err != nil {
		return dto.CategoryResponse{}, notFoundOr(err, "kategori tidak ditemukan")
	}

	return dto.NewCategoryResponse(category), nil
}

func (s *categoryService) Create(ctx context.Context, req dto.CategoryRequest) (dto.CategoryResponse, error) {
	category, err := s.save(ctx, model.Category{}, req)
	if err != nil {
		return dto.CategoryResponse{}, err
	}
	return s.Get(ctx, idRef(category.ID))
}

func (s *categoryService) Update(ctx context.Context, id int64, req dto.CategoryRequest) (dto.CategoryResponse, error) {
	category, err := s.categories.FindByID(ctx, id)
	if err != nil {
		return dto.CategoryResponse{}, notFoundOr(err, "kategori tidak ditemukan")
	}

	if _, err := s.save(ctx, category, req); err != nil {
		return dto.CategoryResponse{}, err
	}
	return s.Get(ctx, idRef(id))
}

// save membuat kategori baru bila category.ID nol, atau memperbaruinya.
func (s *categoryService) save(ctx context.Context, category model.Category, req dto.CategoryRequest) (model.Category, error) {
	req.Normalize()

	if problems := validation.ValidateCategory(req); len(problems) > 0 {
		return model.Category{}, apperr.Validation(problems)
	}

	taken, err := s.categories.NameTaken(ctx, req.Name, category.ID)
	if err != nil {
		return model.Category{}, apperr.Internal(err)
	}
	if taken {
		return model.Category{}, apperr.Validation(map[string]string{"name": "nama kategori sudah dipakai"})
	}

	if category.ID == 0 || category.Name != req.Name {
		base := slug.Make(req.Name, "kategori", 120)
		existing, err := s.categories.SlugsWithPrefix(ctx, base, category.ID)
		if err != nil {
			return model.Category{}, apperr.Internal(err)
		}
		category.Slug = slug.Unique(base, existing)
	}
	category.Name = req.Name
	category.Description = req.Description

	if category.ID == 0 {
		err = s.categories.Create(ctx, &category)
	} else {
		err = s.categories.Update(ctx, &category)
	}
	if errors.Is(err, repository.ErrDuplicate) {
		return model.Category{}, apperr.Conflict("kategori dengan nama atau slug yang sama baru saja dibuat")
	}
	if err != nil {
		return model.Category{}, apperr.Internal(err)
	}

	return category, nil
}

func (s *categoryService) Delete(ctx context.Context, id int64) error {
	err := s.categories.Delete(ctx, id)
	if errors.Is(err, repository.ErrInUse) {
		return apperr.Conflict("kategori masih dipakai article, pindahkan article-nya lebih dulu")
	}
	if err != nil {
		return notFoundOr(err, "kategori tidak ditemukan")
	}
	return nil
}

type TagService interface {
	List(ctx context.Context, query string, p pagination.Params) ([]dto.TagResponse, pagination.Meta, error)
	Create(ctx context.Context, req dto.TagRequest) (dto.TagResponse, error)
	Update(ctx context.Context, id int64, req dto.TagRequest) (dto.TagResponse, error)
	Delete(ctx context.Context, id int64) error
}

type tagService struct {
	tags repository.TagRepository
}

func NewTagService(tags repository.TagRepository) TagService {
	return &tagService{tags: tags}
}

func (s *tagService) List(ctx context.Context, query string, p pagination.Params) ([]dto.TagResponse, pagination.Meta, error) {
	tags, total, err := s.tags.List(ctx, dto.NormalizeTag(query), p)
	if err != nil {
		return nil, pagination.Meta{}, apperr.Internal(err)
	}
	return dto.NewTagResponses(tags), pagination.NewMeta(p, total), nil
}

func (s *tagService) Create(ctx context.Context, req dto.TagRequest) (dto.TagResponse, error) {
	tag, err := s.save(ctx, model.Tag{}, req)
	if err != nil {
		return dto.TagResponse{}, err
	}
	return s.get(ctx, tag.ID)
}

func (s *tagService) Update(ctx context.Context, id int64, req dto.TagRequest) (dto.TagResponse, error) {
	tag, err := s.tags.FindByID(ctx, id)
	if err != nil {
		return dto.TagResponse{}, notFoundOr(err, "tag tidak ditemukan")
	}

	if _, err := s.save(ctx, tag, req); err != nil {
		return dto.TagResponse{}, err
	}
	return s.get(ctx, id)
}

func (s *tagService) save(ctx context.Context, tag model.Tag, req dto.TagRequest) (model.Tag, error) {
	req.Normalize()

	if problems := validation.ValidateTag(req); len(problems) > 0 {
		return model.Tag{}, apperr.Validation(problems)
	}

	taken, err := s.tags.NameTaken(ctx, req.Name, tag.ID)
	if err != nil {
		return model.Tag{}, apperr.Internal(err)
	}
	if taken {
		return model.Tag{}, apperr.Validation(map[string]string{"name": "tag sudah ada"})
	}

	if tag.ID == 0 || tag.Name != req.Name {
		base := slug.Make(req.Name, "tag", 60)
		existing, err := s.tags.SlugsWithPrefix(ctx, base, tag.ID)
		if err != nil {
			return model.Tag{}, apperr.Internal(err)
		}
		tag.Slug = slug.Unique(base, existing)
	}
	tag.Name = req.Name

	if tag.ID == 0 {
		err = s.tags.Create(ctx, &tag)
	} else {
		err = s.tags.Update(ctx, &tag)
	}
	if errors.Is(err, repository.ErrDuplicate) {
		return model.Tag{}, apperr.Conflict("tag dengan nama atau slug yang sama baru saja dibuat")
	}
	if err != nil {
		return model.Tag{}, apperr.Internal(err)
	}

	return tag, nil
}

func (s *tagService) get(ctx context.Context, id int64) (dto.TagResponse, error) {
	tag, err := s.tags.FindByID(ctx, id)
	if err != nil {
		return dto.TagResponse{}, notFoundOr(err, "tag tidak ditemukan")
	}
	return dto.NewTagResponse(tag), nil
}

func (s *tagService) Delete(ctx context.Context, id int64) error {
	if err := s.tags.Delete(ctx, id); err != nil {
		return notFoundOr(err, "tag tidak ditemukan")
	}
	return nil
}

// parseID mengenali ref yang berupa id. Slug dijamin tidak pernah seluruhnya
// angka (lihat slug.Make), jadi keduanya tidak tertukar.
func parseID(ref string) (int64, bool) {
	if !slug.IsNumeric(ref) {
		return 0, false
	}
	id, err := strconv.ParseInt(ref, 10, 64)
	return id, err == nil && id > 0
}

func idRef(id int64) string {
	return strconv.FormatInt(id, 10)
}
