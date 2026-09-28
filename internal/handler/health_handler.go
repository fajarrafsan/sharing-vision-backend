package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"warta/internal/response"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]string{
			"status":   "unavailable",
			"database": "tidak terjangkau",
		})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"database": "terhubung",
	})
}
