package router

import (
	"log"
	"net/http"
	"time"

	"sharing-vision-backend/internal/handler"
)

func New(articles *handler.ArticleHandler, health *handler.HealthHandler, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health.Live)
	mux.HandleFunc("GET /health/live", health.Live)
	mux.HandleFunc("GET /health/ready", health.Ready)

	mux.HandleFunc("POST /article", articles.Create)
	mux.HandleFunc("POST /article/{$}", articles.Create)

	mux.HandleFunc("GET /article/{limit}/{offset}", articles.List)
	mux.HandleFunc("GET /article/{id}", articles.GetByID)

	mux.HandleFunc("POST /article/{id}", articles.Update)
	mux.HandleFunc("PUT /article/{id}", articles.Update)
	mux.HandleFunc("PATCH /article/{id}", articles.Update)

	mux.HandleFunc("DELETE /article/{id}", articles.Delete)
	mux.HandleFunc("POST /article/{id}/delete", articles.Delete)

	return logRequest(cors(mux, allowedOrigin))
}

func cors(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}
