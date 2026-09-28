package handler

import (
	"net/http"
	"strconv"

	"warta/internal/dto"
	"warta/internal/pagination"
	"warta/internal/response"
	"warta/internal/service"
)

type CategoryHandler struct {
	service service.CategoryService
}

func NewCategoryHandler(s service.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: s}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.List(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, categories)
}

func (h *CategoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	category, err := h.service.Get(r.Context(), r.PathValue("ref"))
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, category)
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	category, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/categories/"+strconv.FormatInt(category.ID, 10))
	response.Data(w, http.StatusCreated, category)
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	var req dto.CategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	category, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, category)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

type TagHandler struct {
	service service.TagService
	pages   pagination.Parser
}

func NewTagHandler(s service.TagService, pages pagination.Parser) *TagHandler {
	return &TagHandler{service: s, pages: pages}
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	page, err := h.pages.Parse(r.URL.Query())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	tags, meta, err := h.service.List(r.Context(), queryString(r, "q"), page)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.List(w, tags, meta)
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.TagRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	tag, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusCreated, tag)
}

func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	var req dto.TagRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	tag, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Data(w, http.StatusOK, tag)
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}
