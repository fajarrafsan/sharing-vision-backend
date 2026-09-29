package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"warta/internal/model"
)

type UserTokenRepository interface {
	// Replace mencabut token lama dengan tujuan yang sama lalu menyimpan yang
	// baru, sehingga hanya tautan terakhir yang berlaku.
	Replace(ctx context.Context, t *model.UserToken) error
	FindByHash(ctx context.Context, purpose, hash string) (model.UserToken, error)
	// Use menandai token terpakai. false berarti sudah terpakai lebih dulu.
	Use(ctx context.Context, id int64) (bool, error)
	// IssuedSince memeriksa apakah token dengan tujuan itu sudah dibuat
	// setelah waktu tertentu. Dipakai untuk memberi jeda antar email.
	IssuedSince(ctx context.Context, userID int64, purpose string, since time.Time) (bool, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

type userTokenRepository struct {
	db *sql.DB
}

func NewUserTokenRepository(db *sql.DB) UserTokenRepository {
	return &userTokenRepository{db: db}
}

func (r *userTokenRepository) Replace(ctx context.Context, t *model.UserToken) error {
	return withTx(ctx, r.db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			"UPDATE user_tokens SET used_at = CURRENT_TIMESTAMP WHERE user_id = ? AND purpose = ? AND used_at IS NULL",
			t.UserID, t.Purpose)
		if err != nil {
			return err
		}

		result, err := tx.ExecContext(ctx,
			"INSERT INTO user_tokens (user_id, purpose, token_hash, expires_at) VALUES (?, ?, ?, ?)",
			t.UserID, t.Purpose, t.TokenHash, t.ExpiresAt)
		if err != nil {
			return mapError(err)
		}
		t.ID, err = result.LastInsertId()
		return err
	})
}

func (r *userTokenRepository) FindByHash(ctx context.Context, purpose, hash string) (model.UserToken, error) {
	var t model.UserToken
	var used sql.NullTime
	err := r.db.QueryRowContext(ctx,
		"SELECT id, user_id, purpose, token_hash, expires_at, used_at FROM user_tokens WHERE token_hash = ? AND purpose = ?",
		hash, purpose,
	).Scan(&t.ID, &t.UserID, &t.Purpose, &t.TokenHash, &t.ExpiresAt, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return model.UserToken{}, ErrNotFound
	}
	t.UsedAt = nullTimePtr(used)
	return t, err
}

func (r *userTokenRepository) Use(ctx context.Context, id int64) (bool, error) {
	result, err := r.db.ExecContext(ctx,
		"UPDATE user_tokens SET used_at = CURRENT_TIMESTAMP WHERE id = ? AND used_at IS NULL", id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (r *userTokenRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM user_tokens WHERE expires_at < ?", before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *userTokenRepository) IssuedSince(ctx context.Context, userID int64, purpose string, since time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM user_tokens WHERE user_id = ? AND purpose = ? AND created_at > ?)",
		userID, purpose, since,
	).Scan(&exists)
	return exists, err
}
