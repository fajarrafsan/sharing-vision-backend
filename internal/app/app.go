package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"warta/internal/auth"
	"warta/internal/config"
	"warta/internal/handler"
	"warta/internal/middleware"
	"warta/internal/pagination"
	"warta/internal/repository"
	"warta/internal/router"
	"warta/internal/service"
)

const maxBodyBytes = 1 << 20

// App merangkai repository, service, dan handler menjadi satu http.Handler.
// Dipakai cmd/api dan test end-to-end.
type App struct {
	Handler http.Handler
	Auth    service.AuthService

	refreshTokens repository.RefreshTokenRepository
}

func New(cfg config.Config, db *sql.DB, hasher auth.PasswordHasher) *App {
	pages := pagination.Parser{DefaultPerPage: cfg.DefaultPerPage, MaxPerPage: cfg.MaxPerPage}
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL)

	users := repository.NewUserRepository(db)
	refreshTokens := repository.NewRefreshTokenRepository(db)
	categories := repository.NewCategoryRepository(db)
	tags := repository.NewTagRepository(db)
	articles := repository.NewArticleRepository(db)
	comments := repository.NewCommentRepository(db)

	authService := service.NewAuthService(users, refreshTokens, hasher, tokens, cfg.RefreshTokenTTL)

	handlers := router.Handlers{
		Health:     handler.NewHealthHandler(db),
		Docs:       handler.NewDocsHandler(),
		Auth:       handler.NewAuthHandler(authService),
		Users:      handler.NewUserHandler(service.NewUserService(users), pages),
		Categories: handler.NewCategoryHandler(service.NewCategoryService(categories)),
		Tags:       handler.NewTagHandler(service.NewTagService(tags), pages),
		Articles:   handler.NewArticleHandler(service.NewArticleService(articles, categories), pages),
		Comments:   handler.NewCommentHandler(service.NewCommentService(comments, articles), pages),
	}

	return &App{
		Handler: router.New(handlers, router.Options{
			Tokens:       tokens,
			CORSOrigins:  cfg.CORSOrigins,
			AuthLimiter:  middleware.NewRateLimiter(cfg.AuthRateLimit),
			MaxBodyBytes: maxBodyBytes,
		}),
		Auth:          authService,
		refreshTokens: refreshTokens,
	}
}

// CleanupTokens menghapus refresh token kedaluwarsa secara berkala sampai ctx
// selesai.
func (a *App) CleanupTokens(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deleted, err := a.refreshTokens.DeleteExpired(ctx, time.Now())
			if err != nil {
				slog.Error("gagal membersihkan refresh token", "error", err)
				continue
			}
			if deleted > 0 {
				slog.Info("refresh token kedaluwarsa dihapus", "jumlah", deleted)
			}
		}
	}
}
