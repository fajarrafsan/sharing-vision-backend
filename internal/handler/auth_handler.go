package handler

import (
	"net/http"

	"warta/internal/auth"
	"warta/internal/dto"
	"warta/internal/response"
	"warta/internal/service"
)

type AuthHandler struct {
	service service.AuthService
}

func NewAuthHandler(s service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	tokens, err := h.service.Register(r.Context(), req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusCreated, tokens)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	tokens, err := h.service.Login(r.Context(), req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	tokens, err := h.service.Refresh(r.Context(), req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	if err := h.service.Logout(r.Context(), req); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.Me(r.Context(), auth.ActorFrom(r.Context()))
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, user)
}

func (h *AuthHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), auth.ActorFrom(r.Context()), req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, user)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ChangePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	if err := h.service.ChangePassword(r.Context(), auth.ActorFrom(r.Context()), req); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyEmailRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	user, err := h.service.VerifyEmail(r.Context(), req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, user)
}

func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ResendVerification(r.Context(), auth.ActorFrom(r.Context())); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ForgotPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	if err := h.service.ForgotPassword(r.Context(), req); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ResetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	if err := h.service.ResetPassword(r.Context(), req); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}
