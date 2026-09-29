package handler

import (
	"net/http"

	"warta/internal/auth"
	"warta/internal/dto"
	"warta/internal/pagination"
	"warta/internal/response"
	"warta/internal/service"
)

type CommentHandler struct {
	service service.CommentService
	pages   pagination.Parser
}

func NewCommentHandler(s service.CommentService, pages pagination.Parser) *CommentHandler {
	return &CommentHandler{service: s, pages: pages}
}

func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	articleID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	page, err := h.pages.Parse(r.URL.Query())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	comments, meta, err := h.service.List(r.Context(), auth.ActorFrom(r.Context()), articleID, page)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.List(w, comments, meta)
}

func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	articleID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	var req dto.CommentRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	comment, err := h.service.Create(r.Context(), auth.ActorFrom(r.Context()), articleID, req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusCreated, comment)
}

func (h *CommentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	var req dto.CommentRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	comment, err := h.service.Update(r.Context(), auth.ActorFrom(r.Context()), id, req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, comment)
}

func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	if err := h.service.Delete(r.Context(), auth.ActorFrom(r.Context()), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func (h *CommentHandler) Report(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	var req dto.ReportRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	result, err := h.service.Report(r.Context(), auth.ActorFrom(r.Context()), id, req)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Data(w, http.StatusOK, result)
}

func (h *CommentHandler) ListReported(w http.ResponseWriter, r *http.Request) {
	page, err := h.pages.Parse(r.URL.Query())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	comments, meta, err := h.service.ListReported(r.Context(), page)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.List(w, comments, meta)
}

func (h *CommentHandler) Moderate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	var req dto.ModerationRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	if err := h.service.Moderate(r.Context(), id, req); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}
