// Package storage menyimpan gambar sampul yang diunggah, di disk lokal atau
// di object storage yang kompatibel dengan S3.
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// URLPrefix adalah awalan path publik untuk berkas yang diunggah.
const URLPrefix = "/uploads/"

var (
	ErrUnsupported = errors.New("format gambar harus JPEG, PNG, WebP, atau GIF")
	ErrTooLarge    = errors.New("ukuran gambar melebihi batas")
	ErrNotFound    = errors.New("berkas tidak ditemukan")

	// Nama berkas selalu buatan server: 32 karakter hex dan ekstensi yang dikenal.
	namePattern = regexp.MustCompile(`^[a-f0-9]{32}\.(jpg|png|webp|gif)$`)
)

var extensions = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
}

// ContentType adalah tipe MIME berkas dari ekstensinya.
func ContentType(name string) string {
	ext := name[strings.LastIndex(name, ".")+1:]
	for mime, e := range extensions {
		if e == ext {
			return mime
		}
	}
	return "application/octet-stream"
}

// Store adalah tempat menyimpan gambar. Nama berkas acak dan tidak pernah
// ditimpa, jadi setiap berkas aman di-cache selamanya.
type Store interface {
	// SaveImage memeriksa isi berkas lalu menyimpannya. Hasilnya path publik
	// berkas itu, misalnya /uploads/ab12....png.
	SaveImage(ctx context.Context, r io.Reader) (string, error)
	// Exists memeriksa bahwa url adalah path upload yang sah dan berkasnya ada.
	Exists(ctx context.Context, url string) bool
	// Open membuka berkas untuk disajikan. ErrNotFound bila tidak ada.
	Open(ctx context.Context, name string) (*Object, error)
}

// Object adalah berkas yang dibuka untuk disajikan dengan http.ServeContent.
type Object struct {
	io.ReadSeekCloser
	ModTime time.Time
}

// ValidName memeriksa nama berkas buatan server, sekaligus menolak path
// traversal.
func ValidName(name string) bool {
	return namePattern.MatchString(name)
}

// nameFromURL mengambil nama berkas dari path publik yang sah.
func nameFromURL(url string) (string, bool) {
	name, ok := strings.CutPrefix(url, URLPrefix)
	if !ok || !ValidName(name) {
		return "", false
	}
	return name, true
}

// inspect membaca berkas dan memeriksa isinya (bukan ekstensi atau header
// dari klien). Hasilnya isi berkas dan nama acak untuknya.
func inspect(r io.Reader, maxBytes int64) (data []byte, name string, err error) {
	data, err = io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxBytes {
		return nil, "", ErrTooLarge
	}

	ext, ok := extensions[http.DetectContentType(data)]
	if !ok {
		return nil, "", ErrUnsupported
	}
	// Pustaka standar tidak bisa membaca WebP, jadi WebP cukup dikenali dari
	// signature-nya. Format lain harus benar-benar bisa dibaca sebagai gambar.
	if ext != "webp" {
		if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
			return nil, "", ErrUnsupported
		}
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, "", err
	}
	return data, hex.EncodeToString(b) + "." + ext, nil
}
