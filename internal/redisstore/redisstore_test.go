package redisstore

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// Test ini butuh Redis sungguhan dan dilewati bila TEST_REDIS_URL kosong:
//
//	TEST_REDIS_URL=redis://127.0.0.1:6379/15 go test ./internal/redisstore/
func connect(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL kosong, test Redis dilewati")
	}
	client, err := Connect(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// unique membuat nama batas baru di setiap test supaya tidak saling ganggu.
func unique(t *testing.T) string {
	return fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano())
}

func TestLimiterSharedAcrossInstances(t *testing.T) {
	client := connect(t)
	ctx := context.Background()
	name := unique(t)

	// Dua "instance" memakai nama batas yang sama dan berbagi hitungan.
	a := NewLimiter(client, name, 3)
	b := NewLimiter(client, name, 3)
	for i, l := range []*Limiter{a, b, a} {
		if ok, _ := l.Allow(ctx, "ip:1"); !ok {
			t.Fatalf("permintaan ke-%d seharusnya lolos", i+1)
		}
	}
	ok, wait := b.Allow(ctx, "ip:1")
	if ok || wait <= 0 || wait > 20*time.Second {
		t.Fatalf("permintaan ke-4 seharusnya ditolak dengan waktu tunggu wajar: %v %v", ok, wait)
	}

	// Kunci dan nama batas lain tidak ikut terkena.
	if ok, _ := a.Allow(ctx, "ip:2"); !ok {
		t.Fatal("kunci lain seharusnya lolos")
	}
	if ok, _ := NewLimiter(client, name+"-lain", 3).Allow(ctx, "ip:1"); !ok {
		t.Fatal("batas lain seharusnya terpisah")
	}

	// Kunci kedaluwarsa sendiri supaya Redis tidak penuh.
	ttl := client.PTTL(ctx, keyPrefix+"rl:"+name+":ip:1").Val()
	if ttl <= 0 || ttl > 62*time.Second {
		t.Fatalf("TTL kunci: %v", ttl)
	}
}

func TestLimiterRefills(t *testing.T) {
	client := connect(t)
	ctx := context.Background()

	// 600 per menit = satu token tiap 100 ms.
	l := NewLimiter(client, unique(t), 600)
	for range 600 {
		l.Allow(ctx, "k")
	}
	if ok, _ := l.Allow(ctx, "k"); ok {
		t.Fatal("seharusnya habis")
	}
	time.Sleep(150 * time.Millisecond)
	if ok, _ := l.Allow(ctx, "k"); !ok {
		t.Fatal("token seharusnya terisi kembali")
	}
}

func TestLimiterFailsOpen(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	defer client.Close()
	if ok, _ := NewLimiter(client, "mati", 1).Allow(context.Background(), "k"); !ok {
		t.Fatal("Redis mati seharusnya meloloskan permintaan")
	}
	if !NewDeduper(client, time.Minute).First(context.Background(), "ip:1", 1) {
		t.Fatal("Redis mati seharusnya tetap menghitung bacaan")
	}
}

func TestDeduper(t *testing.T) {
	client := connect(t)
	ctx := context.Background()
	viewer := unique(t)

	a := NewDeduper(client, 200*time.Millisecond)
	b := NewDeduper(client, 200*time.Millisecond)
	if !a.First(ctx, viewer, 1) {
		t.Fatal("bacaan pertama dihitung")
	}
	if b.First(ctx, viewer, 1) {
		t.Fatal("instance lain seharusnya tahu pembaca ini sudah dihitung")
	}
	if !b.First(ctx, viewer, 2) {
		t.Fatal("artikel lain dihitung terpisah")
	}
	time.Sleep(250 * time.Millisecond)
	if !a.First(ctx, viewer, 1) {
		t.Fatal("setelah window lewat, dihitung lagi")
	}
}
