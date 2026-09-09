package router

import (
	"log"
	"net/http"
	"time"

	"sharing-vision-backend/internal/handler"
)

func New(articles *handler.ArticleHandler, health *handler.HealthHandler) http.Handler {
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

	return logRequest(mux)
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}
