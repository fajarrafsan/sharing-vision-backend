// Package storagetest menyediakan server S3 tiruan untuk test.
package storagetest

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"

	"warta/internal/storage"
)

// FakeS3 menjalankan server S3 di memori selama test dan mengembalikan
// konfigurasinya. Bucket "warta" sudah dibuat.
func FakeS3(t testing.TB) storage.S3Config {
	t.Helper()
	backend := s3mem.New()
	if err := backend.CreateBucket("warta"); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(gofakes3.New(backend).Server())
	t.Cleanup(server.Close)

	u, _ := url.Parse(server.URL)
	return storage.S3Config{
		Endpoint:  u.Host,
		Region:    "us-east-1",
		Bucket:    "warta",
		AccessKey: "kunci",
		SecretKey: "rahasia",
		Prefix:    "uploads/",
	}
}
