package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config berlaku untuk AWS S3 dan layanan yang kompatibel: Cloudflare R2,
// MinIO, Backblaze B2, DigitalOcean Spaces, dan sejenisnya.
type S3Config struct {
	// Endpoint tanpa skema, misalnya s3.ap-southeast-1.amazonaws.com atau
	// <akun>.r2.cloudflarestorage.com.
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	// Prefix adalah folder di dalam bucket, misalnya uploads/.
	Prefix string
}

// S3 menyimpan gambar di object storage, sehingga semua instance service
// melihat berkas yang sama. Bucket boleh privat; berkas disajikan lewat
// service di /uploads/{nama}.
type S3 struct {
	client   *minio.Client
	bucket   string
	prefix   string
	maxBytes int64
}

// NewS3 terhubung ke bucket dan memastikan bucket itu ada.
func NewS3(ctx context.Context, cfg S3Config, maxBytes int64) (*S3, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("memeriksa bucket %s: %w", cfg.Bucket, err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket %s tidak ada", cfg.Bucket)
	}
	return &S3{client: client, bucket: cfg.Bucket, prefix: cfg.Prefix, maxBytes: maxBytes}, nil
}

func (s *S3) SaveImage(ctx context.Context, r io.Reader) (string, error) {
	data, name, err := inspect(r, s.maxBytes)
	if err != nil {
		return "", err
	}
	_, err = s.client.PutObject(ctx, s.bucket, s.prefix+name, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType:  ContentType(name),
		CacheControl: "public, max-age=31536000, immutable",
	})
	if err != nil {
		return "", err
	}
	return URLPrefix + name, nil
}

func (s *S3) Exists(ctx context.Context, url string) bool {
	name, ok := nameFromURL(url)
	if !ok {
		return false
	}
	_, err := s.client.StatObject(ctx, s.bucket, s.prefix+name, minio.StatObjectOptions{})
	return err == nil
}

func (s *S3) Open(ctx context.Context, name string) (*Object, error) {
	if !ValidName(name) {
		return nil, ErrNotFound
	}
	obj, err := s.client.GetObject(ctx, s.bucket, s.prefix+name, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	// GetObject baru menghubungi server saat dibaca; Stat memastikan berkasnya ada.
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		if minio.ToErrorResponse(err).Code == minio.NoSuchKey {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &Object{ReadSeekCloser: obj, ModTime: info.LastModified}, nil
}

// Ping memastikan bucket masih terjangkau, untuk /health/ready.
func (s *S3) Ping(ctx context.Context) error {
	_, err := s.client.BucketExists(ctx, s.bucket)
	return err
}
