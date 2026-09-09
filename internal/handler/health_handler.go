package handler

import (
	"database/sql"
	"net/http"

	"sharing-vision-backend/internal/response"
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
	if err := h.db.Ping(); err != nil {
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
