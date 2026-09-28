package handler

import (
	"net/http"

	"warta/internal/auth"
	"warta/internal/dto"
	"warta/internal/pagination"
	"warta/internal/response"
	"warta/internal/service"
)

type UserHandler struct {
	service service.UserService
	pages   pagination.Parser
}

func NewUserHandler(s service.UserService, pages pagination.Parser) *UserHandler {
	return &UserHandler{service: s, pages: pages}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	page, err := h.pages.Parse(r.URL.Query())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	query := dto.UserQuery{Query: queryString(r, "q"), Role: queryString(r, "role")}
	users, meta, err := h.service.List(r.Context(), query, page)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.List(w, users, meta)
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	user, err := h.service.Get(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, user)
}

func (h *UserHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	var req dto.UpdateRoleRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	user, err := h.service.UpdateRole(r.Context(), auth.ActorFrom(r.Context()), id, req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, user)
}
