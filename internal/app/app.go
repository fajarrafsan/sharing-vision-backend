package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"warta/internal/auth"
	"warta/internal/clientip"
	"warta/internal/config"
	"warta/internal/handler"
	"warta/internal/middleware"
	"warta/internal/pagination"
	"warta/internal/repository"
	"warta/internal/router"
	"warta/internal/service"
	"warta/internal/storage"
)

const maxBodyBytes = 1 << 20

// App merangkai repository, service, dan handler menjadi satu http.Handler.
// Dipakai cmd/api dan test end-to-end.
type App struct {
	Handler http.Handler
	Auth    service.AuthService

	refreshTokens repository.RefreshTokenRepository
}

func New(cfg config.Config, db *sql.DB, hasher auth.PasswordHasher) (*App, error) {
	uploads, err := storage.NewLocal(cfg.UploadDir, cfg.MaxUploadBytes)
	if err != nil {
		return nil, err
	}
	resolver, err := clientip.NewResolver(cfg.TrustedProxies)
	if err != nil {
		return nil, err
	}

	pages := pagination.Parser{DefaultPerPage: cfg.DefaultPerPage, MaxPerPage: cfg.MaxPerPage}
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL)

	users := repository.NewUserRepository(db)
	refreshTokens := repository.NewRefreshTokenRepository(db)
	categories := repository.NewCategoryRepository(db)
	tags := repository.NewTagRepository(db)
	articles := repository.NewArticleRepository(db)
	comments := repository.NewCommentRepository(db)
	engagement := repository.NewEngagementRepository(db)

	authService := service.NewAuthService(users, refreshTokens, hasher, tokens, cfg.RefreshTokenTTL)

	handlers := router.Handlers{
		Health:     handler.NewHealthHandler(db),
		Docs:       handler.NewDocsHandler(),
		Auth:       handler.NewAuthHandler(authService),
		Users:      handler.NewUserHandler(service.NewUserService(users), pages),
		Categories: handler.NewCategoryHandler(service.NewCategoryService(categories)),
		Tags:       handler.NewTagHandler(service.NewTagService(tags), pages),
		Articles:   handler.NewArticleHandler(service.NewArticleService(articles, categories, engagement, uploads), pages),
		Comments:   handler.NewCommentHandler(service.NewCommentService(comments, articles, cfg.CommentHideThreshold), pages),
		Stats:      handler.NewStatsHandler(service.NewStatsService(repository.NewStatsRepository(db))),
		Uploads:    handler.NewUploadHandler(uploads),
	}

	return &App{
		Handler: router.New(handlers, router.Options{
			Tokens:         tokens,
			CORSOrigins:    cfg.CORSOrigins,
			AuthLimiter:    middleware.NewRateLimiter(cfg.AuthRateLimit),
			CommentLimiter: middleware.NewRateLimiter(cfg.CommentRateLimit),
			UploadLimiter:  middleware.NewRateLimiter(cfg.UploadRateLimit),
			ClientIP:       resolver,
			MaxBodyBytes:   maxBodyBytes,
			MaxUploadBytes: cfg.MaxUploadBytes + 64<<10,
		}),
		Auth:          authService,
		refreshTokens: refreshTokens,
	}, nil
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
