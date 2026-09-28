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

// UserService dipakai admin untuk mengelola akun.
type UserService interface {
	List(ctx context.Context, q dto.UserQuery, p pagination.Params) ([]dto.UserResponse, pagination.Meta, error)
	Get(ctx context.Context, id int64) (dto.UserResponse, error)
	UpdateRole(ctx context.Context, actor auth.Actor, id int64, req dto.UpdateRoleRequest) (dto.UserResponse, error)
}

type userService struct {
	users repository.UserRepository
}

func NewUserService(users repository.UserRepository) UserService {
	return &userService{users: users}
}

func (s *userService) List(ctx context.Context, q dto.UserQuery, p pagination.Params) ([]dto.UserResponse, pagination.Meta, error) {
	if q.Role != "" && !model.Role(q.Role).Valid() {
		return nil, pagination.Meta{}, apperr.BadRequest("role harus admin, author, atau reader")
	}

	users, total, err := s.users.List(ctx, repository.UserFilter{Query: q.Query, Role: model.Role(q.Role)}, p)
	if err != nil {
		return nil, pagination.Meta{}, apperr.Internal(err)
	}

	return dto.NewUserResponses(users), pagination.NewMeta(p, total), nil
}

func (s *userService) Get(ctx context.Context, id int64) (dto.UserResponse, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return dto.UserResponse{}, notFoundOr(err, "user tidak ditemukan")
	}
	return dto.NewUserResponse(user), nil
}

func (s *userService) UpdateRole(ctx context.Context, actor auth.Actor, id int64, req dto.UpdateRoleRequest) (dto.UserResponse, error) {
	req.Normalize()

	if problems := validation.ValidateRole(req); len(problems) > 0 {
		return dto.UserResponse{}, apperr.Validation(problems)
	}

	// Admin tidak bisa menurunkan dirinya sendiri, sehingga selalu tersisa
	// minimal satu admin.
	if actor.ID == id {
		return dto.UserResponse{}, apperr.Forbidden("tidak bisa mengubah role akun sendiri")
	}

	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return dto.UserResponse{}, notFoundOr(err, "user tidak ditemukan")
	}

	if err := s.users.UpdateRole(ctx, id, model.Role(req.Role)); err != nil {
		return dto.UserResponse{}, apperr.Internal(err)
	}

	return s.Get(ctx, user.ID)
}

// notFoundOr mengubah ErrNotFound menjadi 404 dengan pesan yang diberikan, dan
// error lain menjadi 500.
func notFoundOr(err error, message string) error {
	if errors.Is(err, repository.ErrNotFound) {
		return apperr.NotFound(message)
	}
	return apperr.Internal(err)
}
