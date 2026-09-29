package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"

	"warta/internal/clientip"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

// Nilai contoh di docker-compose.yml dan README. Boleh dipakai untuk mencoba
// di komputer sendiri, tapi ditolak saat APP_ENV=production.
var (
	placeholderSecrets = map[string]bool{
		"ganti-dengan-rahasia-acak-minimal-32-karakter":     true,
		"dev-secret-yang-panjangnya-lebih-dari-32-karakter": true,
	}
	weakPasswords = map[string]bool{
		"": true, "root": true, "admin": true, "admin123": true, "admin12345": true,
		"password": true, "password123": true, "rahasia123": true, "12345678": true,
	}
)

type Config struct {
	// Env adalah development atau production. Production menolak nilai
	// bawaan yang tidak aman.
	Env         string
	AppPort     string
	AutoMigrate bool
	LogFormat   string
	LogLevel    string
	CORSOrigins []string
	// TrustedProxies adalah IP atau CIDR reverse proxy yang boleh mengisi
	// X-Forwarded-For. Kosong berarti service diakses langsung.
	TrustedProxies []string

	DefaultPerPage int
	MaxPerPage     int

	JWTSecret       string
	JWTIssuer       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// AuthRateLimit adalah jumlah permintaan per menit per IP ke endpoint
	// register, login, dan refresh.
	AuthRateLimit int
	// CommentRateLimit adalah jumlah komentar dan laporan per menit per akun.
	CommentRateLimit int
	// UploadRateLimit adalah jumlah upload gambar per menit per akun.
	UploadRateLimit int
	// CommentHideThreshold adalah jumlah laporan dari akun berbeda yang
	// membuat komentar disembunyikan otomatis sampai ditinjau admin.
	CommentHideThreshold int

	// UploadStorage adalah local (folder UploadDir) atau s3.
	UploadStorage string
	// UploadDir adalah folder gambar sampul yang diunggah.
	UploadDir      string
	MaxUploadBytes int64
	S3Endpoint     string
	S3Region       string
	S3Bucket       string
	S3AccessKey    string
	S3SecretKey    string
	S3UseSSL       bool
	S3Prefix       string

	// RedisURL kosong berarti rate limit dan hitungan dibaca disimpan di
	// memori tiap instance. Isi bila service dijalankan beberapa instance.
	RedisURL string

	AdminName     string
	AdminEmail    string
	AdminPassword string

	// AppURL adalah alamat frontend. Dipakai untuk tautan di email, sitemap,
	// dan RSS.
	AppURL string
	// RequireEmailVerification membuat akun yang emailnya belum terverifikasi
	// tidak bisa berkomentar atau melaporkan komentar.
	RequireEmailVerification bool
	// SMTP kosong berarti email hanya dicatat ke log (development).
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	MailFrom     string

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
		Env:            l.oneOf("APP_ENV", EnvDevelopment, EnvDevelopment, EnvProduction),
		AppPort:        l.str("APP_PORT", "8080"),
		AutoMigrate:    l.boolean("AUTO_MIGRATE", true),
		LogFormat:      l.oneOf("LOG_FORMAT", "text", "text", "json"),
		LogLevel:       l.oneOf("LOG_LEVEL", "info", "debug", "info", "warn", "error"),
		CORSOrigins:    l.list("CORS_ORIGINS", "*"),
		TrustedProxies: l.list("TRUSTED_PROXIES", ""),

		DefaultPerPage: l.integer("DEFAULT_PER_PAGE", 10),
		MaxPerPage:     l.integer("MAX_PER_PAGE", 100),

		JWTSecret:       l.str("JWT_SECRET", ""),
		JWTIssuer:       l.str("JWT_ISSUER", "warta"),
		AccessTokenTTL:  l.duration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: l.duration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		AuthRateLimit:   l.integer("AUTH_RATE_LIMIT", 20),

		CommentRateLimit:     l.integer("COMMENT_RATE_LIMIT", 5),
		UploadRateLimit:      l.integer("UPLOAD_RATE_LIMIT", 10),
		CommentHideThreshold: l.integer("COMMENT_HIDE_THRESHOLD", 3),

		UploadStorage:  l.oneOf("UPLOAD_STORAGE", "local", "local", "s3"),
		UploadDir:      l.str("UPLOAD_DIR", "uploads"),
		MaxUploadBytes: int64(l.integer("MAX_UPLOAD_MB", 2)) << 20,
		S3Endpoint:     l.str("S3_ENDPOINT", ""),
		S3Region:       l.str("S3_REGION", ""),
		S3Bucket:       l.str("S3_BUCKET", ""),
		S3AccessKey:    l.str("S3_ACCESS_KEY", ""),
		S3SecretKey:    l.str("S3_SECRET_KEY", ""),
		S3UseSSL:       l.boolean("S3_USE_SSL", true),
		S3Prefix:       l.str("S3_PREFIX", "uploads/"),

		RedisURL: l.str("REDIS_URL", ""),

		AdminName:     l.str("ADMIN_NAME", "Administrator"),
		AdminEmail:    strings.ToLower(l.str("ADMIN_EMAIL", "")),
		AdminPassword: l.str("ADMIN_PASSWORD", ""),

		AppURL:       strings.TrimRight(l.str("APP_URL", "http://localhost:5173"), "/"),
		SMTPHost:     l.str("SMTP_HOST", ""),
		SMTPPort:     l.str("SMTP_PORT", "587"),
		SMTPUsername: l.str("SMTP_USERNAME", ""),
		SMTPPassword: l.str("SMTP_PASSWORD", ""),
		MailFrom:     l.str("MAIL_FROM", "Warta <noreply@warta.local>"),

		DBHost:     l.str("DB_HOST", "127.0.0.1"),
		DBPort:     l.str("DB_PORT", "3306"),
		DBUser:     l.str("DB_USER", "root"),
		DBPassword: l.str("DB_PASSWORD", ""),
		DBName:     l.str("DB_NAME", "warta"),
	}

	// Verifikasi email wajib di production, opsional di development supaya
	// bisa mencoba tanpa SMTP.
	cfg.RequireEmailVerification = l.boolean("REQUIRE_EMAIL_VERIFICATION", cfg.Env == EnvProduction)

	cfg.validate(&l)
	return cfg, l.err()
}

func (c Config) validate(l *loader) {
	if len(c.JWTSecret) < 32 {
		l.fail("JWT_SECRET wajib diisi, minimal 32 karakter")
	}
	if c.DefaultPerPage < 1 || c.MaxPerPage < c.DefaultPerPage {
		l.fail("DEFAULT_PER_PAGE minimal 1 dan tidak boleh melebihi MAX_PER_PAGE")
	}
	if c.MaxUploadBytes < 1<<20 || c.MaxUploadBytes > 20<<20 {
		l.fail("MAX_UPLOAD_MB harus antara 1 dan 20")
	}
	for key, value := range map[string]int{
		"AUTH_RATE_LIMIT":        c.AuthRateLimit,
		"COMMENT_RATE_LIMIT":     c.CommentRateLimit,
		"UPLOAD_RATE_LIMIT":      c.UploadRateLimit,
		"COMMENT_HIDE_THRESHOLD": c.CommentHideThreshold,
	} {
		if value < 1 {
			l.fail("%s minimal 1", key)
		}
	}
	if _, err := clientip.NewResolver(c.TrustedProxies); err != nil {
		l.fail("TRUSTED_PROXIES: %v", err)
	}
	if u, err := url.Parse(c.AppURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		l.fail("APP_URL harus alamat lengkap frontend, misalnya https://warta.id")
	}
	if c.SMTPHost != "" {
		if _, err := strconv.Atoi(c.SMTPPort); err != nil {
			l.fail("SMTP_PORT harus berupa angka")
		}
		if (c.SMTPUsername == "") != (c.SMTPPassword == "") {
			l.fail("SMTP_USERNAME dan SMTP_PASSWORD harus diisi berdua atau dikosongkan berdua")
		}
	}
	if c.UploadStorage == "s3" {
		for key, value := range map[string]string{
			"S3_ENDPOINT": c.S3Endpoint, "S3_BUCKET": c.S3Bucket,
			"S3_ACCESS_KEY": c.S3AccessKey, "S3_SECRET_KEY": c.S3SecretKey,
		} {
			if value == "" {
				l.fail("%s wajib diisi bila UPLOAD_STORAGE=s3", key)
			}
		}
		if strings.Contains(c.S3Endpoint, "://") {
			l.fail("S3_ENDPOINT ditulis tanpa http:// atau https://; atur TLS dengan S3_USE_SSL")
		}
	}
	if c.RedisURL != "" && !strings.HasPrefix(c.RedisURL, "redis://") && !strings.HasPrefix(c.RedisURL, "rediss://") {
		l.fail("REDIS_URL harus diawali redis:// atau rediss://")
	}
	if (c.AdminEmail == "") != (c.AdminPassword == "") {
		l.fail("ADMIN_EMAIL dan ADMIN_PASSWORD harus diisi berdua atau dikosongkan berdua")
	}
	if c.AdminPassword != "" && (len(c.AdminPassword) < 8 || len(c.AdminPassword) > 72) {
		l.fail("ADMIN_PASSWORD harus 8 sampai 72 karakter")
	}

	if c.Env != EnvProduction {
		return
	}
	if placeholderSecrets[c.JWTSecret] {
		l.fail("JWT_SECRET masih nilai contoh; buat yang acak, misalnya: openssl rand -base64 48")
	}
	if c.AdminPassword != "" && (len(c.AdminPassword) < 12 || weakPasswords[strings.ToLower(c.AdminPassword)]) {
		l.fail("ADMIN_PASSWORD terlalu lemah untuk production, minimal 12 karakter dan bukan nilai contoh")
	}
	if weakPasswords[strings.ToLower(c.DBPassword)] {
		l.fail("DB_PASSWORD kosong atau masih nilai contoh")
	}
	for _, origin := range c.CORSOrigins {
		if origin == "*" {
			l.fail("CORS_ORIGINS tidak boleh * di production; isi dengan origin frontend")
		}
	}
	if !strings.HasPrefix(c.AppURL, "https://") {
		l.fail("APP_URL harus https di production")
	}
	if c.SMTPHost == "" {
		l.fail("SMTP_HOST wajib di production supaya email verifikasi dan reset password terkirim")
	}
}

// Warnings adalah hal yang tidak menghentikan service tapi sebaiknya
// diperbaiki. Dicatat ke log saat service menyala.
func (c Config) Warnings() []string {
	if c.Env == EnvProduction {
		return nil
	}

	var warnings []string
	if placeholderSecrets[c.JWTSecret] {
		warnings = append(warnings, "JWT_SECRET masih nilai contoh; hanya aman untuk mencoba di komputer sendiri")
	}
	if weakPasswords[strings.ToLower(c.AdminPassword)] && c.AdminPassword != "" {
		warnings = append(warnings, "ADMIN_PASSWORD masih nilai contoh; ganti sebelum dipakai orang lain")
	}
	if c.SMTPHost == "" {
		warnings = append(warnings, "SMTP_HOST kosong; email verifikasi dan reset password hanya dicatat ke log")
	}
	return warnings
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
