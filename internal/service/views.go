package service

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// ViewDeduper menjawab apakah pembaca baru pertama kali membaca artikel itu
// dalam rentang waktu tertentu, supaya memuat ulang halaman hanya dihitung
// sekali.
type ViewDeduper interface {
	First(ctx context.Context, viewer string, articleID int64) bool
}

// viewDeduper menyimpan ingatan itu di memori. Cukup untuk satu instance;
// untuk beberapa instance pakai redisstore.Deduper.
type viewDeduper struct {
	mu        sync.Mutex
	window    time.Duration
	seen      map[string]time.Time
	lastSweep time.Time
	now       func() time.Time
}

// ViewWindow adalah rentang waktu satu pembaca dihitung sekali per artikel.
const ViewWindow = 30 * time.Minute

func newViewDeduper(window time.Duration) *viewDeduper {
	return &viewDeduper{window: window, seen: make(map[string]time.Time), now: time.Now}
}

func (d *viewDeduper) First(_ context.Context, viewer string, articleID int64) bool {
	return d.first(viewer, articleID, d.now())
}

// first mengembalikan true bila pembaca belum membaca artikel ini dalam window.
func (d *viewDeduper) first(viewer string, articleID int64, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if now.Sub(d.lastSweep) > d.window {
		for key, at := range d.seen {
			if now.Sub(at) > d.window {
				delete(d.seen, key)
			}
		}
		d.lastSweep = now
	}

	key := viewer + "|" + strconv.FormatInt(articleID, 10)
	if at, ok := d.seen[key]; ok && now.Sub(at) <= d.window {
		return false
	}
	d.seen[key] = now
	return true
}
