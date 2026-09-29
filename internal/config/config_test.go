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
	if len(c.Warnings()) != 2 {
		t.Fatalf("seharusnya ada peringatan: %v", c.Warnings())
	}
}

func TestProductionRejectsUnsafeValues(t *testing.T) {
	c := valid()
	c.Env = EnvProduction
	c.JWTSecret = "ganti-dengan-rahasia-acak-minimal-32-karakter"

	p := problems(c)
	for _, want := range []string{"JWT_SECRET", "ADMIN_PASSWORD", "DB_PASSWORD", "CORS_ORIGINS"} {
		if !strings.Contains(p, want) {
			t.Errorf("%s seharusnya ditolak: %s", want, p)
		}
	}

	c.JWTSecret = "Zq8v0k3Hw1Rj6Yb2Nf5Tc9Lm4Px7Ds0Ga3Ve8Ku1Oi6"
	c.AdminPassword = "Kuat-dan-panjang-2026"
	c.DBPassword = "db-rahasia-yang-acak"
	c.CORSOrigins = []string{"https://warta.id"}
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
