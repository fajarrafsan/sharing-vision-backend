package config

import (
	"strings"
	"testing"
)

func valid() Config {
	return Config{
		Env:                  EnvDevelopment,
		JWTSecret:            "rahasia-acak-yang-panjangnya-lebih-dari-32",
		DefaultPerPage:       10,
		MaxPerPage:           100,
		MaxUploadBytes:       2 << 20,
		AuthRateLimit:        20,
		CommentRateLimit:     5,
		UploadRateLimit:      10,
		CommentHideThreshold: 3,
		CORSOrigins:          []string{"*"},
		AdminEmail:           "admin@warta.local",
		AdminPassword:        "admin12345",
		DBPassword:           "root",
		AppURL:               "http://localhost:5173",
		SMTPPort:             "587",
		UploadStorage:        "local",
	}
}

func problems(c Config) string {
	var l loader
	c.validate(&l)
	return strings.Join(l.problems, "; ")
}

func TestDevelopmentAllowsExampleValues(t *testing.T) {
	c := valid()
	c.JWTSecret = "ganti-dengan-rahasia-acak-minimal-32-karakter"
	if p := problems(c); p != "" {
		t.Fatalf("mode development seharusnya menerima nilai contoh: %s", p)
	}
	if len(c.Warnings()) != 3 {
		t.Fatalf("seharusnya ada peringatan: %v", c.Warnings())
	}
}

func TestProductionRejectsUnsafeValues(t *testing.T) {
	c := valid()
	c.Env = EnvProduction
	c.JWTSecret = "ganti-dengan-rahasia-acak-minimal-32-karakter"

	p := problems(c)
	for _, want := range []string{"JWT_SECRET", "ADMIN_PASSWORD", "DB_PASSWORD", "CORS_ORIGINS", "APP_URL", "SMTP_HOST"} {
		if !strings.Contains(p, want) {
			t.Errorf("%s seharusnya ditolak: %s", want, p)
		}
	}

	c.JWTSecret = "Zq8v0k3Hw1Rj6Yb2Nf5Tc9Lm4Px7Ds0Ga3Ve8Ku1Oi6"
	c.AdminPassword = "Kuat-dan-panjang-2026"
	c.DBPassword = "db-rahasia-yang-acak"
	c.CORSOrigins = []string{"https://warta.id"}
	c.AppURL = "https://warta.id"
	c.SMTPHost = "smtp.example.com"
	c.SMTPUsername = "warta"
	c.SMTPPassword = "smtp-rahasia"
	if p := problems(c); p != "" {
		t.Fatalf("konfigurasi production yang aman ditolak: %s", p)
	}
	if len(c.Warnings()) != 0 {
		t.Fatalf("production tidak memberi peringatan: %v", c.Warnings())
	}
}

func TestTrustedProxiesValidated(t *testing.T) {
	c := valid()
	c.TrustedProxies = []string{"10.0.0.0/8", "bukan-ip"}
	if p := problems(c); !strings.Contains(p, "TRUSTED_PROXIES") {
		t.Fatalf("TRUSTED_PROXIES yang salah seharusnya ditolak: %q", p)
	}
}

func TestAppURLValidated(t *testing.T) {
	c := valid()
	c.AppURL = "warta.id"
	if p := problems(c); !strings.Contains(p, "APP_URL") {
		t.Fatalf("APP_URL tanpa skema seharusnya ditolak: %q", p)
	}
}

func TestSharedStorageValidated(t *testing.T) {
	c := valid()
	c.UploadStorage = "s3"
	c.S3Endpoint = "https://s3.example.com"
	c.RedisURL = "localhost:6379"
	p := problems(c)
	for _, want := range []string{"S3_BUCKET", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_ENDPOINT ditulis tanpa", "REDIS_URL"} {
		if !strings.Contains(p, want) {
			t.Errorf("%s seharusnya ditolak: %s", want, p)
		}
	}

	c.S3Endpoint = "s3.example.com"
	c.S3Bucket = "warta"
	c.S3AccessKey = "kunci"
	c.S3SecretKey = "rahasia"
	c.RedisURL = "redis://redis:6379/0"
	if p := problems(c); p != "" {
		t.Fatalf("konfigurasi yang benar ditolak: %s", p)
	}
}
