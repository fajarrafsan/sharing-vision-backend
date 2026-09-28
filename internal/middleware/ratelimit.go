package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"warta/internal/apperr"
	"warta/internal/response"
)

// RateLimiter adalah token bucket per kunci (alamat IP) yang disimpan di
// memori. Cukup untuk satu instance; bila service dijalankan beberapa
// instance, batasnya berlaku per instance.
type RateLimiter struct {
	mu        sync.Mutex
	rate      float64 // token per detik
	burst     float64
	buckets   map[string]*bucket
	now       func() time.Time
	lastSweep time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter mengizinkan perMinute permintaan per menit per kunci, dengan
// ledakan sebanyak perMinute sekaligus.
func NewRateLimiter(perMinute int) *RateLimiter {
	return &RateLimiter{
		rate:    float64(perMinute) / 60,
		burst:   float64(perMinute),
		buckets: make(map[string]*bucket),
		now:     time.Now,
	}
}

// Allow mengambil satu token. Bila habis, retryAfter adalah waktu tunggu
// sampai token berikutnya tersedia.
func (l *RateLimiter) Allow(key string) (ok bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, found := l.buckets[key]
	if !found {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}

	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now

	if b.tokens < 1 {
		wait := (1 - b.tokens) / l.rate
		return false, time.Duration(wait * float64(time.Second))
	}

	b.tokens--
	return true, 0
}

// sweep membuang bucket yang sudah penuh kembali, supaya map tidak tumbuh
// terus oleh IP yang hanya lewat sekali.
func (l *RateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now

	full := time.Duration(l.burst / l.rate * float64(time.Second))
	for key, b := range l.buckets {
		if now.Sub(b.last) > full {
			delete(l.buckets, key)
		}
	}
}

func RateLimit(l *RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, retryAfter := l.Allow(clientIP(r))
			if !ok {
				seconds := int(math.Ceil(retryAfter.Seconds()))
				w.Header().Set("Retry-After", strconv.Itoa(max(seconds, 1)))
				response.Error(w, r, apperr.TooManyRequests("terlalu banyak percobaan, coba lagi sebentar lagi"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP memakai alamat koneksi langsung. Header seperti X-Forwarded-For
// sengaja tidak dipercaya karena bisa diisi sembarangan oleh klien.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
