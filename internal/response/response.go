package response

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"sharing-vision-backend/internal/apperr"
)

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Println("gagal menulis response:", err)
	}
}

func Empty(w http.ResponseWriter, status int) {
	JSON(w, status, map[string]string{})
}

func Error(w http.ResponseWriter, err error) {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		log.Println("error tidak terduga:", err)
		JSON(w, http.StatusInternalServerError, map[string]string{
			"message": "terjadi kesalahan pada server",
		})
		return
	}

	if appErr.Cause != nil {
		log.Println("error:", appErr.Cause)
	}

	JSON(w, appErr.Status, appErr)
}
