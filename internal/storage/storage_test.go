package storage

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"
)

func pngBytes(t *testing.T) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestSaveImage(t *testing.T) {
	s, err := NewLocal(t.TempDir(), 1024)
	if err != nil {
		t.Fatal(err)
	}

	url, err := s.SaveImage(bytes.NewReader(pngBytes(t)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, URLPrefix) || !strings.HasSuffix(url, ".png") || !s.Exists(url) {
		t.Fatalf("url: %s", url)
	}

	// Teks yang menyamar sebagai gambar dan PNG rusak ditolak.
	for _, data := range [][]byte{[]byte("<script>alert(1)</script>"), append([]byte("\x89PNG\r\n\x1a\n"), 1, 2, 3)} {
		if _, err := s.SaveImage(bytes.NewReader(data)); !errors.Is(err, ErrUnsupported) {
			t.Errorf("seharusnya ditolak: %v", err)
		}
	}

	if _, err := s.SaveImage(bytes.NewReader(make([]byte, 2048))); !errors.Is(err, ErrTooLarge) {
		t.Errorf("berkas besar: %v", err)
	}
}

func TestExists(t *testing.T) {
	s, _ := NewLocal(t.TempDir(), 1024)
	for _, url := range []string{"", "/uploads/", "/uploads/../go.mod", "/lain/abc.png", "/uploads/" + strings.Repeat("a", 32) + ".png"} {
		if s.Exists(url) {
			t.Errorf("%q seharusnya tidak ada", url)
		}
	}
}
