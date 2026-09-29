package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"warta/internal/app"
	"warta/internal/auth"
	"warta/internal/storage/storagetest"
)

// TestMultiInstance menjalankan dua instance service yang berbagi database,
// Redis, dan object storage, seperti di balik load balancer. Butuh MySQL dan
// Redis:
//
//	TEST_DB_HOST=127.0.0.1 TEST_DB_PASSWORD=root TEST_REDIS_URL=redis://127.0.0.1:6379/15 go test ./internal/app/
func TestMultiInstance(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("TEST_REDIS_URL kosong, test beberapa instance dilewati")
	}

	cfg := testConfig(t)
	migrate(t, cfg, latestVersion)
	db := connect(t, cfg)

	s3 := storagetest.FakeS3(t)
	cfg.UploadStorage = "s3"
	cfg.S3Endpoint, cfg.S3Region, cfg.S3Bucket = s3.Endpoint, s3.Region, s3.Bucket
	cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Prefix = s3.AccessKey, s3.SecretKey, s3.Prefix
	cfg.S3UseSSL = false
	cfg.RedisURL = redisURL
	cfg.AuthRateLimit = 3
	cfg.TrustedProxies = []string{"127.0.0.1", "::1"}

	instances := make([]client, 2)
	for i := range instances {
		a, err := app.New(cfg, db, auth.BcryptHasher{Cost: bcrypt.MinCost}, &outbox{})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { a.Close() })
		if i == 0 {
			if err := a.Auth.EnsureAdmin(context.Background(), "Admin", "admin@warta.test", "admin12345"); err != nil {
				t.Fatal(err)
			}
		}
		server := httptest.NewServer(a.Handler)
		t.Cleanup(server.Close)
		instances[i] = client{t: t, base: server.URL}
	}
	a, b := instances[0], instances[1]

	var ready map[string]string
	if err := json.Unmarshal(a.do("GET", "/health/ready", "", nil).body, &ready); err != nil {
		t.Fatal(err)
	}
	if ready["redis"] != "terhubung" || ready["storage"] != "terhubung" || ready["database"] != "terhubung" {
		t.Fatalf("ready: %v", ready)
	}

	// Kunci Redis dipakai bersama oleh semua run test, jadi setiap run memakai
	// IP pengunjung sendiri (alamat dokumentasi IPv6).
	run := time.Now().UnixNano() & 0xffff
	attackerIP := fmt.Sprintf("2001:db8::%x:1", run)
	adminIP := fmt.Sprintf("2001:db8::%x:2", run)
	readerIP := fmt.Sprintf("2001:db8::%x:3", run)

	// Batas login dihitung bersama: percobaan yang tersebar ke dua instance
	// tetap terhitung sebagai satu.
	wrong := map[string]string{"email": "admin@warta.test", "password": "salah-sekali"}
	a.doFrom(attackerIP, "POST", "/api/v1/auth/login", "", wrong).expectError(http.StatusUnauthorized, "unauthorized")
	b.doFrom(attackerIP, "POST", "/api/v1/auth/login", "", wrong).expectError(http.StatusUnauthorized, "unauthorized")
	a.doFrom(attackerIP, "POST", "/api/v1/auth/login", "", wrong).expectError(http.StatusUnauthorized, "unauthorized")
	b.doFrom(attackerIP, "POST", "/api/v1/auth/login", "", wrong).expectError(http.StatusTooManyRequests, "too_many_requests")

	var admin tokens
	a.doFrom(adminIP, "POST", "/api/v1/auth/login", "", map[string]string{"email": "admin@warta.test", "password": "admin12345"}).
		expect(http.StatusOK, &admin)

	// Gambar yang diunggah lewat satu instance bisa dibuka dan dipakai lewat
	// instance lain.
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 3, 3)))
	var uploaded struct{ URL string }
	a.upload(admin.AccessToken, buf.Bytes()).expect(http.StatusCreated, &uploaded)
	res := b.do("GET", uploaded.URL, "", nil).expect(http.StatusOK, nil)
	if !bytes.Equal(res.body, buf.Bytes()) || res.header.Get("Content-Type") != "image/png" {
		t.Fatalf("berkas dari instance lain: %s", res.header.Get("Content-Type"))
	}
	if res.header.Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatalf("cache: %s", res.header.Get("Cache-Control"))
	}
	b.do("GET", "/uploads/"+fmt.Sprintf("%032d", 0)+".png", "", nil).expectError(http.StatusNotFound, "not_found")

	var cat category
	b.do("POST", "/api/v1/categories", admin.AccessToken, map[string]string{"name": "Teknologi"}).expect(http.StatusCreated, &cat)
	var post article
	b.do("POST", "/api/v1/articles", admin.AccessToken, map[string]any{
		"title": "Tulisan yang dibaca dari dua instance", "content": content,
		"category_id": cat.ID, "status": "published", "cover_image": uploaded.URL,
	}).expect(http.StatusCreated, &post)

	// Pembaca yang sama dihitung sekali walau permintaannya dilayani instance
	// yang berbeda.
	view := fmt.Sprintf("/api/v1/articles/%d/view", post.ID)
	a.doFrom(readerIP, "POST", view, "", nil).expect(http.StatusNoContent, nil)
	b.doFrom(readerIP, "POST", view, "", nil).expect(http.StatusNoContent, nil)
	var seen article
	a.do("GET", fmt.Sprintf("/api/v1/articles/%d", post.ID), "", nil).expect(http.StatusOK, &seen)
	if seen.ViewCount != 1 {
		t.Fatalf("view_count %d, ingin 1", seen.ViewCount)
	}
}
