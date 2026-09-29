package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound = errors.New("data tidak ditemukan")
	// ErrDuplicate berarti nilai unik (email, nama, slug) sudah dipakai.
	ErrDuplicate = errors.New("data sudah ada")
	// ErrInUse berarti baris masih dirujuk tabel lain sehingga tidak bisa dihapus.
	ErrInUse = errors.New("data masih dipakai")
	// ErrMissingReference berarti baris yang dirujuk (misalnya kategori) tidak ada.
	ErrMissingReference = errors.New("data yang dirujuk tidak ada")
)

// Kode error MySQL. 1216 dan 1217 adalah versi lama dari 1452 dan 1451 yang
// masih bisa muncul di beberapa versi server.
const (
	errDuplicateEntry        = 1062
	errRowIsReferenced       = 1451
	errNoReferencedRow       = 1452
	errNoReferencedRowLegacy = 1216
	errRowIsReferencedLegacy = 1217
)

// mapError menerjemahkan kode error MySQL yang punya arti bagi service.
func mapError(err error) error {
	var me *mysql.MySQLError
	if !errors.As(err, &me) {
		return err
	}
	switch me.Number {
	case errDuplicateEntry:
		return ErrDuplicate
	case errRowIsReferenced, errRowIsReferencedLegacy:
		return ErrInUse
	case errNoReferencedRow, errNoReferencedRowLegacy:
		return ErrMissingReference
	}
	return err
}

// DBTX dipenuhi *sql.DB maupun *sql.Tx.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func withTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// slugsWithPrefix mengambil slug yang sama dengan base atau berbentuk base-N,
// bahan untuk slug.Unique. table selalu konstanta dari kode, bukan input.
func slugsWithPrefix(ctx context.Context, q DBTX, table, base string, excludeID int64) ([]string, error) {
	rows, err := q.QueryContext(ctx,
		"SELECT slug FROM "+table+" WHERE (slug = ? OR slug LIKE ?) AND id <> ?",
		base, escapeLike(base)+"-%", excludeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		slugs = append(slugs, s)
	}
	return slugs, rows.Err()
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func whereClause(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(conditions, " AND ")
}
