package storage_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"

	"warta/internal/storage"
	"warta/internal/storage/storagetest"
)

func pngBytes(t *testing.T) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// stores menguji perilaku yang sama untuk semua implementasi.
func stores(t *testing.T, maxBytes int64) map[string]storage.Store {
	local, err := storage.NewLocal(t.TempDir(), maxBytes)
	if err != nil {
		t.Fatal(err)
	}
	s3, err := storage.NewS3(context.Background(), storagetest.FakeS3(t), maxBytes)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]storage.Store{"local": local, "s3": s3}
}

func TestSaveAndOpen(t *testing.T) {
	ctx := context.Background()
	for kind, s := range stores(t, 1024) {
		t.Run(kind, func(t *testing.T) {
			data := pngBytes(t)
			url, err := s.SaveImage(ctx, bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(url, storage.URLPrefix) || !strings.HasSuffix(url, ".png") || !s.Exists(ctx, url) {
				t.Fatalf("url: %s", url)
			}

			obj, err := s.Open(ctx, strings.TrimPrefix(url, storage.URLPrefix))
			if err != nil {
				t.Fatal(err)
			}
			got, _ := io.ReadAll(obj)
			obj.Close()
			if !bytes.Equal(got, data) || obj.ModTime.IsZero() {
				t.Fatalf("isi berkas berbeda (%d byte) atau tanpa waktu", len(got))
			}

			// Teks yang menyamar sebagai gambar dan PNG rusak ditolak.
			for _, bad := range [][]byte{[]byte("<script>alert(1)</script>"), append([]byte("\x89PNG\r\n\x1a\n"), 1, 2, 3)} {
				if _, err := s.SaveImage(ctx, bytes.NewReader(bad)); !errors.Is(err, storage.ErrUnsupported) {
					t.Errorf("seharusnya ditolak: %v", err)
				}
			}
			if _, err := s.SaveImage(ctx, bytes.NewReader(make([]byte, 2048))); !errors.Is(err, storage.ErrTooLarge) {
				t.Errorf("berkas besar: %v", err)
			}
		})
	}
}

func TestMissing(t *testing.T) {
	ctx := context.Background()
	missing := strings.Repeat("a", 32) + ".png"
	for kind, s := range stores(t, 1024) {
		t.Run(kind, func(t *testing.T) {
			for _, url := range []string{"", "/uploads/", "/uploads/../go.mod", "/lain/abc.png", storage.URLPrefix + missing} {
				if s.Exists(ctx, url) {
					t.Errorf("%q seharusnya tidak ada", url)
				}
			}
			for _, name := range []string{missing, "../go.mod", "abc.png"} {
				if _, err := s.Open(ctx, name); !errors.Is(err, storage.ErrNotFound) {
					t.Errorf("Open(%q): %v", name, err)
				}
			}
		})
	}
}

func TestS3MissingBucket(t *testing.T) {
	cfg := storagetest.FakeS3(t)
	cfg.Bucket = "tidak-ada"
	if _, err := storage.NewS3(context.Background(), cfg, 1024); err == nil {
		t.Fatal("bucket yang tidak ada seharusnya ditolak saat menyala")
	}
}
