package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"sharing-vision-backend/internal/apperr"
	"sharing-vision-backend/internal/dto"
	"sharing-vision-backend/internal/response"
	"sharing-vision-backend/internal/service"
)

type ArticleHandler struct {
	service service.ArticleService
}

func NewArticleHandler(s service.ArticleService) *ArticleHandler {
	return &ArticleHandler{service: s}
}

func (h *ArticleHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := readRequest(w, r)
	if !ok {
		return
	}

	article, err := h.service.Create(req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, article)
}

func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(r.PathValue("limit"))
	if err != nil {
		response.Error(w, apperr.BadRequest("limit harus berupa angka"))
		return
	}

	offset, err := strconv.Atoi(r.PathValue("offset"))
	if err != nil {
		response.Error(w, apperr.BadRequest("offset harus berupa angka"))
		return
	}

	articles, err := h.service.List(limit, offset)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, articles)
}

func (h *ArticleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}

	article, err := h.service.GetByID(id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, article)
}

func (h *ArticleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}

	req, ok := readRequest(w, r)
	if !ok {
		return
	}

	article, err := h.service.Update(id, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, article)
}

func (h *ArticleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(id); err != nil {
		response.Error(w, err)
		return
	}

	response.Empty(w, http.StatusOK)
}

func readID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, apperr.BadRequest("id harus berupa angka bulat positif"))
		return 0, false
	}
	return id, true
}

func readRequest(w http.ResponseWriter, r *http.Request) (dto.ArticleRequest, bool) {
	var req dto.ArticleRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperr.BadRequest("body JSON tidak bisa dibaca"))
		return req, false
	}

	return req, true
}
