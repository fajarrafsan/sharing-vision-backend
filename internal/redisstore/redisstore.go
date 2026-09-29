// Package redisstore menyimpan keadaan yang harus sama di semua instance
// service: batas permintaan (rate limit) dan pembaca yang sudah dihitung.
// Tanpa Redis, keduanya disimpan di memori tiap instance.
package redisstore

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// keyPrefix memisahkan kunci Warta dari aplikasi lain di Redis yang sama.
const keyPrefix = "warta:"

// Connect membaca URL seperti redis://:password@host:6379/0 (atau rediss://
// untuk TLS) dan memastikan Redis terjangkau.
func Connect(ctx context.Context, url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}

// tokenBucket menjalankan token bucket secara atomik di Redis. Waktu diambil
// dari jam Redis, bukan jam tiap instance, supaya semua instance sepakat.
// Hasilnya {1, 0} bila lolos, atau {0, tunggu dalam milidetik}.
var tokenBucket = redis.NewScript(`
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)

local state = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(state[1]) or burst
local ts = tonumber(state[2]) or now
tokens = math.min(burst, tokens + math.max(0, now - ts) * rate)

local allowed, wait = 0, 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
else
  wait = math.ceil((1 - tokens) / rate)
end

redis.call('HSET', KEYS[1], 'tokens', tokens, 'ts', now)
redis.call('PEXPIRE', KEYS[1], math.ceil(burst / rate) + 1000)
return {allowed, wait}
`)

// Limiter adalah token bucket yang dihitung bersama oleh semua instance.
type Limiter struct {
	client *redis.Client
	name   string
	rate   float64 // token per milidetik
	burst  int
}

// NewLimiter mengizinkan perMinute permintaan per menit per kunci. name
// membedakan batas yang berbeda, misalnya auth dan comment.
func NewLimiter(client *redis.Client, name string, perMinute int) *Limiter {
	return &Limiter{client: client, name: name, rate: float64(perMinute) / 60000, burst: perMinute}
}

// Allow gagal terbuka: bila Redis bermasalah, permintaan tetap dilayani dan
// masalahnya dicatat, karena menolak semua login lebih buruk daripada
// sementara tanpa batas.
func (l *Limiter) Allow(ctx context.Context, key string) (bool, time.Duration) {
	result, err := tokenBucket.Run(ctx, l.client,
		[]string{keyPrefix + "rl:" + l.name + ":" + key},
		strconv.FormatFloat(l.rate, 'g', -1, 64), l.burst,
	).Int64Slice()
	if err != nil || len(result) != 2 {
		slog.WarnContext(ctx, "rate limit Redis gagal, permintaan diloloskan", "limiter", l.name, "error", err)
		return true, 0
	}
	return result[0] == 1, time.Duration(result[1]) * time.Millisecond
}

// Deduper mengingat pembaca yang sudah dihitung, bersama untuk semua instance.
type Deduper struct {
	client *redis.Client
	window time.Duration
}

func NewDeduper(client *redis.Client, window time.Duration) *Deduper {
	return &Deduper{client: client, window: window}
}

// First bernilai true bila pembaca belum membaca artikel ini dalam window.
// Bila Redis bermasalah, bacaan tetap dihitung.
func (d *Deduper) First(ctx context.Context, viewer string, articleID int64) bool {
	key := keyPrefix + "view:" + strconv.FormatInt(articleID, 10) + ":" + viewer
	first, err := d.client.SetNX(ctx, key, 1, d.window).Result()
	if err != nil {
		slog.WarnContext(ctx, "dedupe Redis gagal, bacaan tetap dihitung", "error", err)
		return true
	}
	return first
}
