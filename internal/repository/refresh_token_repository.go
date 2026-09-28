package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"warta/internal/model"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, t *model.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (model.RefreshToken, error)
	// Revoke mengembalikan false bila token sudah dicabut lebih dulu, misalnya
	// oleh permintaan refresh lain yang berjalan bersamaan.
	Revoke(ctx context.Context, id int64) (bool, error)
	RevokeAllForUser(ctx context.Context, userID int64) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

type refreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, t *model.RefreshToken) error {
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?, ?, ?)",
		t.UserID, t.TokenHash, t.ExpiresAt)
	if err != nil {
		return mapError(err)
	}

	t.ID, err = result.LastInsertId()
	return err
}

func (r *refreshTokenRepository) FindByHash(ctx context.Context, hash string) (model.RefreshToken, error) {
	var t model.RefreshToken
	var revokedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		"SELECT id, user_id, token_hash, expires_at, revoked_at, created_at FROM refresh_tokens WHERE token_hash = ?",
		hash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &revokedAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.RefreshToken{}, ErrNotFound
	}
	if err != nil {
		return model.RefreshToken{}, err
	}

	t.RevokedAt = nullTimePtr(revokedAt)
	return t, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id int64) (bool, error) {
	result, err := r.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked_at = CURRENT_TIMESTAMP WHERE id = ? AND revoked_at IS NULL", id)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	return affected == 1, err
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked_at = CURRENT_TIMESTAMP WHERE user_id = ? AND revoked_at IS NULL", userID)
	return err
}

func (r *refreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM refresh_tokens WHERE expires_at < ?", before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
