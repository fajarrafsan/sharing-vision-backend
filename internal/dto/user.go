package dto

import (
	"strings"
	"time"

	"warta/internal/model"
)

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *RegisterRequest) Normalize() {
	r.Name = collapseSpaces(r.Name)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type TokenResponse struct {
	AccessToken  string       `json:"access_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

type UpdateProfileRequest struct {
	Name string `json:"name"`
}

func (r *UpdateProfileRequest) Normalize() {
	r.Name = collapseSpaces(r.Name)
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type UpdateRoleRequest struct {
	Role string `json:"role"`
}

func (r *UpdateRoleRequest) Normalize() {
	r.Role = strings.ToLower(strings.TrimSpace(r.Role))
}

type UserQuery struct {
	Query string
	Role  string
}

// UserResponse berisi email, jadi hanya untuk pemilik akun dan admin.
type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUserResponse(u model.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func NewUserResponses(users []model.User) []UserResponse {
	responses := make([]UserResponse, 0, len(users))
	for _, u := range users {
		responses = append(responses, NewUserResponse(u))
	}
	return responses
}

// AuthorResponse adalah identitas publik penulis article atau komentar.
type AuthorResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
