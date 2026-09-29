package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"warta/internal/apperr"
	"warta/internal/auth"
	"warta/internal/clientip"
	"warta/internal/dto"
	"warta/internal/response"
	"warta/internal/service"
	"warta/internal/storage"
)

// SetLike menangani PUT (suka) dan DELETE (batal suka).
func (h *ArticleHandler) SetLike(w http.ResponseWriter, r *http.Request) {
	h.engage(w, r, h.service.SetLike)
}

// SetBookmark menangani PUT (simpan) dan DELETE (batal simpan).
func (h *ArticleHandler) SetBookmark(w http.ResponseWriter, r *http.Request) {
	h.engage(w, r, h.service.SetBookmark)
}

func (h *ArticleHandler) engage(
	w http.ResponseWriter,
	r *http.Request,
	set func(ctx context.Context, actor auth.Actor, id int64, on bool) (dto.Engagement, error),
) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	state, err := set(r.Context(), auth.ActorFrom(r.Context()), id, r.Method == http.MethodPut)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Data(w, http.StatusOK, state)
}

func (h *ArticleHandler) RecordView(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, r, err)
		return
	}

	actor := auth.ActorFrom(r.Context())
	viewer := "ip:" + clientip.From(r)
	if actor.Authenticated() {
		viewer = "user:" + strconv.FormatInt(actor.ID, 10)
	}

	if err := h.service.RecordView(r.Context(), actor, id, viewer); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}

func (h *ArticleHandler) ListBookmarks(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, h.service.ListBookmarks)
}

type StatsHandler struct {
	service service.StatsService
}

func NewStatsHandler(s service.StatsService) *StatsHandler {
	return &StatsHandler{service: s}
}

func (h *StatsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	days := service.DefaultStatsDays
	if raw := queryString(r, "days"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			response.Error(w, r, apperr.BadRequest("days harus berupa angka"))
			return
		}
		days = n
	}

	stats, err := h.service.Overview(r.Context(), auth.ActorFrom(r.Context()), days)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Data(w, http.StatusOK, stats)
}

// UploadHandler menerima dan menyajikan gambar sampul.
type UploadHandler struct {
	store storage.Store
}

func NewUploadHandler(store storage.Store) *UploadHandler {
	return &UploadHandler{store: store}
}

// Create menerima multipart/form-data dengan field "image".
func (h *UploadHandler) Create(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("image")
	if err != nil {
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			response.Error(w, r, apperr.PayloadTooLarge("gambar melebihi batas ukuran"))
			return
		}
		response.Error(w, r, apperr.Validation(map[string]string{"image": "kirim gambar di field image (multipart/form-data)"}))
		return
	}
	defer file.Close()

	url, err := h.store.SaveImage(r.Context(), file)
	switch {
	case errors.Is(err, storage.ErrTooLarge):
		response.Error(w, r, apperr.PayloadTooLarge("gambar melebihi batas ukuran"))
		return
	case errors.Is(err, storage.ErrUnsupported):
		response.Error(w, r, apperr.Validation(map[string]string{"image": err.Error()}))
		return
	case err != nil:
		response.Error(w, r, apperr.Internal(err))
		return
	}

	w.Header().Set("Location", url)
	response.Data(w, http.StatusCreated, map[string]string{"url": url})
}

func (h *UploadHandler) Serve(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	obj, err := h.store.Open(r.Context(), name)
	if errors.Is(err, storage.ErrNotFound) {
		response.Error(w, r, apperr.NotFound("berkas tidak ditemukan"))
		return
	}
	if err != nil {
		response.Error(w, r, apperr.Internal(err))
		return
	}
	defer obj.Close()

	// Nama berkas acak dan tidak pernah ditimpa, jadi aman di-cache lama.
	w.Header().Set("Content-Type", storage.ContentType(name))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	http.ServeContent(w, r, name, obj.ModTime, obj)
}
