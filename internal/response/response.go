package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"warta/internal/apperr"
)

type envelope struct {
	Data any `json:"data"`
	Meta any `json:"meta,omitempty"`
}

// JSON menulis body apa adanya. Endpoint API memakai Data atau List supaya
// bentuk response-nya seragam.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("gagal menulis response", "error", err)
	}
}

func Data(w http.ResponseWriter, status int, data any) {
	JSON(w, status, envelope{Data: data})
}

func List(w http.ResponseWriter, data any, meta any) {
	JSON(w, http.StatusOK, envelope{Data: data, Meta: meta})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		appErr = apperr.Internal(err)
	}

	if appErr.Status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "permintaan gagal",
			"method", r.Method, "path", r.URL.Path, "error", appErr.Cause)
	}

	JSON(w, appErr.Status, map[string]any{"error": appErr})
}
