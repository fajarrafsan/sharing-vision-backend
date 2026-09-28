package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string
	AutoMigrate bool
	LogFormat   string
	LogLevel    string
	CORSOrigins []string

	DefaultPerPage int
	MaxPerPage     int

	JWTSecret       string
	JWTIssuer       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// AuthRateLimit adalah jumlah permintaan per menit per IP ke endpoint
	// register, login, dan refresh.
	AuthRateLimit int

	AdminName     string
	AdminEmail    string
	AdminPassword string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

// Load membaca konfigurasi dari environment (dan .env bila ada). Semua nilai
// yang salah dikumpulkan dan dikembalikan sekaligus.
func Load() (Config, error) {
	_ = godotenv.Load()

	var l loader
	cfg := Config{
		AppPort:     l.str("APP_PORT", "8080"),
		AutoMigrate: l.boolean("AUTO_MIGRATE", true),
		LogFormat:   l.oneOf("LOG_FORMAT", "text", "text", "json"),
		LogLevel:    l.oneOf("LOG_LEVEL", "info", "debug", "info", "warn", "error"),
		CORSOrigins: l.list("CORS_ORIGINS", "*"),

		DefaultPerPage: l.integer("DEFAULT_PER_PAGE", 10),
		MaxPerPage:     l.integer("MAX_PER_PAGE", 100),

		JWTSecret:       l.str("JWT_SECRET", ""),
		JWTIssuer:       l.str("JWT_ISSUER", "warta"),
		AccessTokenTTL:  l.duration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: l.duration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		AuthRateLimit:   l.integer("AUTH_RATE_LIMIT", 20),

		AdminName:     l.str("ADMIN_NAME", "Administrator"),
		AdminEmail:    strings.ToLower(l.str("ADMIN_EMAIL", "")),
		AdminPassword: l.str("ADMIN_PASSWORD", ""),

		DBHost:     l.str("DB_HOST", "127.0.0.1"),
		DBPort:     l.str("DB_PORT", "3306"),
		DBUser:     l.str("DB_USER", "root"),
		DBPassword: l.str("DB_PASSWORD", ""),
		DBName:     l.str("DB_NAME", "warta"),
	}

	if len(cfg.JWTSecret) < 32 {
		l.fail("JWT_SECRET wajib diisi, minimal 32 karakter")
	}
	if cfg.DefaultPerPage < 1 || cfg.MaxPerPage < cfg.DefaultPerPage {
		l.fail("DEFAULT_PER_PAGE minimal 1 dan tidak boleh melebihi MAX_PER_PAGE")
	}
	if cfg.AuthRateLimit < 1 {
		l.fail("AUTH_RATE_LIMIT minimal 1")
	}
	if (cfg.AdminEmail == "") != (cfg.AdminPassword == "") {
		l.fail("ADMIN_EMAIL dan ADMIN_PASSWORD harus diisi berdua atau dikosongkan berdua")
	}
	if cfg.AdminPassword != "" && (len(cfg.AdminPassword) < 8 || len(cfg.AdminPassword) > 72) {
		l.fail("ADMIN_PASSWORD harus 8 sampai 72 karakter")
	}

	return cfg, l.err()
}

// DSN dipakai service. multiStatements sengaja tidak dinyalakan di sini.
func (c Config) DSN() string {
	return c.dsn(c.DBName, false)
}

// MigrationDSN dipakai golang-migrate, yang butuh beberapa statement dalam
// satu berkas SQL.
func (c Config) MigrationDSN() string {
	return c.dsn(c.DBName, true)
}

// ServerDSN tidak memilih database, dipakai untuk membuat database-nya.
func (c Config) ServerDSN() string {
	return c.dsn("", false)
}

func (c Config) dsn(dbName string, multiStatements bool) string {
	m := mysql.NewConfig()
	m.User = c.DBUser
	m.Passwd = c.DBPassword
	m.Net = "tcp"
	m.Addr = net.JoinHostPort(c.DBHost, c.DBPort)
	m.DBName = dbName
	m.ParseTime = true
	m.Loc = time.UTC
	m.Collation = "utf8mb4_unicode_ci"
	m.MultiStatements = multiStatements
	// Semua waktu disimpan dan dibaca dalam UTC, tidak bergantung zona
	// waktu server MySQL.
	m.Params = map[string]string{"time_zone": "'+00:00'"}
	return m.FormatDSN()
}

type loader struct {
	problems []string
}

func (l *loader) fail(format string, args ...any) {
	l.problems = append(l.problems, fmt.Sprintf(format, args...))
}

func (l *loader) err() error {
	if len(l.problems) == 0 {
		return nil
	}
	return errors.New("konfigurasi tidak valid: " + strings.Join(l.problems, "; "))
}

func (l *loader) str(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func (l *loader) integer(key string, fallback int) int {
	raw := l.str(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		l.fail("%s harus berupa angka", key)
		return fallback
	}
	return value
}

func (l *loader) boolean(key string, fallback bool) bool {
	raw := l.str(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		l.fail("%s harus true atau false", key)
		return fallback
	}
	return value
}

func (l *loader) duration(key string, fallback time.Duration) time.Duration {
	raw := l.str(key, "")
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		l.fail("%s harus berupa durasi positif, misalnya 15m atau 168h", key)
		return fallback
	}
	return value
}

func (l *loader) oneOf(key, fallback string, allowed ...string) string {
	value := strings.ToLower(l.str(key, fallback))
	for _, a := range allowed {
		if value == a {
			return value
		}
	}
	l.fail("%s harus salah satu dari %s", key, strings.Join(allowed, ", "))
	return fallback
}

func (l *loader) list(key, fallback string) []string {
	var values []string
	for _, part := range strings.Split(l.str(key, fallback), ",") {
		if part = strings.TrimSpace(part); part != "" {
			values = append(values, part)
		}
	}
	return values
}
