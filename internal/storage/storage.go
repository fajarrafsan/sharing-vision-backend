package storage

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

// URLPrefix adalah awalan path publik untuk berkas yang diunggah.
const URLPrefix = "/uploads/"

var (
	ErrUnsupported = errors.New("format gambar harus JPEG, PNG, WebP, atau GIF")
	ErrTooLarge    = errors.New("ukuran gambar melebihi batas")

	// Nama berkas selalu buatan server: 32 karakter hex dan ekstensi yang dikenal.
	namePattern = regexp.MustCompile(`^[a-f0-9]{32}\.(jpg|png|webp|gif)$`)
)

var extensions = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
}

// Local menyimpan gambar di folder lokal.
type Local struct {
	Dir      string
	MaxBytes int64
}

func NewLocal(dir string, maxBytes int64) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Local{Dir: dir, MaxBytes: maxBytes}, nil
}

// SaveImage memeriksa isi berkas (bukan ekstensi atau header dari klien),
// lalu menyimpannya dengan nama acak. Hasilnya path publik berkas itu.
func (s *Local) SaveImage(r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, s.MaxBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > s.MaxBytes {
		return "", ErrTooLarge
	}

	ext, ok := extensions[http.DetectContentType(data)]
	if !ok {
		return "", ErrUnsupported
	}
	// Pustaka standar tidak bisa membaca WebP, jadi WebP cukup dikenali dari
	// signature-nya. Format lain harus benar-benar bisa dibaca sebagai gambar.
	if ext != "webp" {
		if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
			return "", ErrUnsupported
		}
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	name := hex.EncodeToString(b) + "." + ext

	if err := os.WriteFile(filepath.Join(s.Dir, name), data, 0o644); err != nil {
		return "", err
	}
	return URLPrefix + name, nil
}

// Path mengembalikan lokasi berkas di disk untuk nama yang sah.
func (s *Local) Path(name string) (string, bool) {
	if !namePattern.MatchString(name) {
		return "", false
	}
	return filepath.Join(s.Dir, name), true
}

// Exists memeriksa bahwa url adalah path upload yang sah dan berkasnya ada.
func (s *Local) Exists(url string) bool {
	if len(url) <= len(URLPrefix) || url[:len(URLPrefix)] != URLPrefix {
		return false
	}
	path, ok := s.Path(url[len(URLPrefix):])
	if !ok {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
