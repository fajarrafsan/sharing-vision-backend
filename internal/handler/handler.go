package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"warta/internal/apperr"
)

// decodeJSON membaca body JSON ke dst dan menerjemahkan kegagalannya menjadi
// pesan yang jelas bagi klien.
func decodeJSON(r *http.Request, dst any) error {
	err := json.NewDecoder(r.Body).Decode(dst)
	if err == nil {
		return nil
	}

	var maxBytes *http.MaxBytesError
	var typeErr *json.UnmarshalTypeError

	switch {
	case errors.As(err, &maxBytes):
		return apperr.PayloadTooLarge("body melebihi batas ukuran")
	case errors.Is(err, io.EOF):
		return apperr.BadRequest("body JSON wajib diisi")
	case errors.As(err, &typeErr) && typeErr.Field != "":
		return apperr.BadRequest("field " + typeErr.Field + " bertipe salah")
	default:
		return apperr.BadRequest("body JSON tidak bisa dibaca")
	}
}

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.BadRequest(name + " harus berupa angka bulat positif")
	}
	return id, nil
}

func queryString(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

func queryID(r *http.Request, key string) (int64, error) {
	raw := queryString(r, key)
	if raw == "" {
		return 0, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.BadRequest(key + " harus berupa angka bulat positif")
	}
	return id, nil
}
