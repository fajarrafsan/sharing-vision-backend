package repository

import (
	"context"
	"database/sql"
	"time"
)

// EngagementRepository menyimpan interaksi pembaca: suka, bookmark, dan
// jumlah dibaca.
type EngagementRepository interface {
	SetLike(ctx context.Context, userID, articleID int64, liked bool) error
	SetBookmark(ctx context.Context, userID, articleID int64, bookmarked bool) error
	// State mengembalikan apakah user sudah menyukai dan menyimpan artikel.
	State(ctx context.Context, userID, articleID int64) (liked, bookmarked bool, err error)
	RecordView(ctx context.Context, articleID int64, day time.Time) error
}

type engagementRepository struct {
	db *sql.DB
}

func NewEngagementRepository(db *sql.DB) EngagementRepository {
	return &engagementRepository{db: db}
}

func (r *engagementRepository) SetLike(ctx context.Context, userID, articleID int64, liked bool) error {
	return r.toggle(ctx, "article_likes", userID, articleID, liked)
}

func (r *engagementRepository) SetBookmark(ctx context.Context, userID, articleID int64, bookmarked bool) error {
	return r.toggle(ctx, "bookmarks", userID, articleID, bookmarked)
}

// toggle bersifat idempoten: menyukai dua kali tetap satu suka. table selalu
// konstanta dari kode.
func (r *engagementRepository) toggle(ctx context.Context, table string, userID, articleID int64, on bool) error {
	var err error
	if on {
		_, err = r.db.ExecContext(ctx, "INSERT IGNORE INTO "+table+" (user_id, article_id) VALUES (?, ?)", userID, articleID)
	} else {
		_, err = r.db.ExecContext(ctx, "DELETE FROM "+table+" WHERE user_id = ? AND article_id = ?", userID, articleID)
	}
	return mapError(err)
}

func (r *engagementRepository) State(ctx context.Context, userID, articleID int64) (bool, bool, error) {
	var liked, bookmarked bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM article_likes WHERE user_id = ? AND article_id = ?),
		       EXISTS (SELECT 1 FROM bookmarks WHERE user_id = ? AND article_id = ?)`,
		userID, articleID, userID, articleID,
	).Scan(&liked, &bookmarked)
	return liked, bookmarked, err
}

func (r *engagementRepository) RecordView(ctx context.Context, articleID int64, day time.Time) error {
	return withTx(ctx, r.db, func(tx *sql.Tx) error {
		// updated_at = updated_at mencegah ON UPDATE mengubah waktu ubah artikel
		// hanya karena dibaca.
		_, err := tx.ExecContext(ctx,
			"UPDATE articles SET view_count = view_count + 1, updated_at = updated_at WHERE id = ?", articleID)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO article_daily_views (article_id, day, views) VALUES (?, ?, 1)
			ON DUPLICATE KEY UPDATE views = views + 1`,
			articleID, day.Format(time.DateOnly))
		return mapError(err)
	})
}
