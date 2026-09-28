package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"warta/migrations"
)

func Connect(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// CreateDatabase dijalankan sebelum migrasi karena golang-migrate hanya bisa
// mengisi database, tidak membuatnya.
func CreateDatabase(ctx context.Context, serverDSN, name string) error {
	db, err := sql.Open("mysql", serverDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", quoteIdent(name)))
	return err
}

func quoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func MigrateUp(dsn, name string) error {
	return run(dsn, name, func(m *migrate.Migrate) error { return m.Up() })
}

func MigrateDown(dsn, name string) error {
	return run(dsn, name, func(m *migrate.Migrate) error { return m.Down() })
}

// MigrateTo naik atau turun sampai versi tertentu.
func MigrateTo(dsn, name string, version uint) error {
	return run(dsn, name, func(m *migrate.Migrate) error { return m.Migrate(version) })
}

// Version mengembalikan versi migrasi saat ini. dirty berarti migrasi terakhir
// gagal di tengah jalan dan perlu diperbaiki manual.
func Version(dsn, name string) (version uint, dirty bool, err error) {
	err = run(dsn, name, func(m *migrate.Migrate) error {
		version, dirty, err = m.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			return nil
		}
		return err
	})
	return version, dirty, err
}

func run(dsn, name string, fn func(m *migrate.Migrate) error) error {
	m, err := newMigrate(dsn, name)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := fn(m); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func newMigrate(dsn, name string) (*migrate.Migrate, error) {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	driver, err := migratemysql.WithInstance(db, &migratemysql.Config{DatabaseName: name})
	if err != nil {
		db.Close()
		return nil, err
	}

	return migrate.NewWithInstance("iofs", source, "mysql", driver)
}
