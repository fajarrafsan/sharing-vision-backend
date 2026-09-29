package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"warta/internal/response"
)

// Check memeriksa satu ketergantungan, misalnya Redis atau object storage.
type Check func(ctx context.Context) error

type HealthHandler struct {
	db     *sql.DB
	checks map[string]Check
}

// NewHealthHandler selalu memeriksa database, ditambah checks bila ada.
func NewHealthHandler(db *sql.DB, checks map[string]Check) *HealthHandler {
	return &HealthHandler{db: db, checks: checks}
}

func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready bernilai 503 bila salah satu ketergantungan tidak terjangkau, supaya
// load balancer berhenti mengirim permintaan ke instance ini.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]Check{"database": h.db.PingContext}
	for name, check := range h.checks {
		checks[name] = check
	}

	body := map[string]string{"status": "ok"}
	status := http.StatusOK
	for name, check := range checks {
		if err := check(ctx); err != nil {
			body[name] = "tidak terjangkau"
			body["status"] = "unavailable"
			status = http.StatusServiceUnavailable
			continue
		}
		body[name] = "terhubung"
	}
	response.JSON(w, status, body)
}
