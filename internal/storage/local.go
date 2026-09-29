package storage

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Local menyimpan gambar di folder lokal. Untuk beberapa instance di server
// berbeda, pakai S3 supaya semua instance melihat berkas yang sama.
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

func (s *Local) SaveImage(_ context.Context, r io.Reader) (string, error) {
	data, name, err := inspect(r, s.MaxBytes)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(s.Dir, name), data, 0o644); err != nil {
		return "", err
	}
	return URLPrefix + name, nil
}

func (s *Local) Exists(_ context.Context, url string) bool {
	name, ok := nameFromURL(url)
	if !ok {
		return false
	}
	info, err := os.Stat(filepath.Join(s.Dir, name))
	return err == nil && info.Mode().IsRegular()
}

func (s *Local) Open(_ context.Context, name string) (*Object, error) {
	if !ValidName(name) {
		return nil, ErrNotFound
	}
	f, err := os.Open(filepath.Join(s.Dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, ErrNotFound
	}
	return &Object{ReadSeekCloser: f, ModTime: info.ModTime()}, nil
}
