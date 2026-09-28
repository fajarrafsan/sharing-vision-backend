package middleware

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"warta/internal/apperr"
	"warta/internal/auth"
	"warta/internal/model"
	"warta/internal/response"
)

// Authenticate membaca access token dari header Authorization bila ada dan
// menaruh Actor di context. Tanpa header, permintaan diteruskan sebagai
// anonim. Token yang dikirim tapi tidak valid langsung ditolak, supaya klien
// tahu harus refresh alih-alih diam-diam diperlakukan sebagai anonim.
func Authenticate(tokens *auth.TokenManager) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				next.ServeHTTP(w, r)
				return
			}

			scheme, token, ok := strings.Cut(header, " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
				unauthorized(w, r, "header Authorization harus berbentuk: Bearer <token>")
				return
			}

			actor, err := tokens.Parse(strings.TrimSpace(token))
			if errors.Is(err, auth.ErrExpiredToken) {
				unauthorized(w, r, "access token kedaluwarsa, lakukan refresh")
				return
			}
			if err != nil {
				unauthorized(w, r, "access token tidak valid")
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithActor(r.Context(), actor)))
		})
	}
}

func unauthorized(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="warta"`)
	response.Error(w, r, apperr.Unauthorized(message))
}

// RequireAuth menolak permintaan tanpa access token.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth.ActorFrom(r.Context()).Authenticated() {
			unauthorized(w, r, "butuh access token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole menolak pengguna yang role-nya tidak ada di daftar. Dipasang
// setelah RequireAuth.
func RequireRole(roles ...model.Role) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !slices.Contains(roles, auth.ActorFrom(r.Context()).Role) {
				response.Error(w, r, apperr.Forbidden("role kamu tidak punya akses ke endpoint ini"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
