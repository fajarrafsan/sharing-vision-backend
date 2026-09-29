package service

import (
	"strconv"
	"sync"
	"time"
)

// viewDeduper mengingat pasangan pembaca dan artikel selama window, supaya
// satu pembaca yang memuat ulang halaman hanya dihitung sekali. Disimpan di
// memori, jadi bila service dijalankan beberapa instance, dedupe berlaku per
// instance.
type viewDeduper struct {
	mu        sync.Mutex
	window    time.Duration
	seen      map[string]time.Time
	lastSweep time.Time
}

func newViewDeduper(window time.Duration) *viewDeduper {
	return &viewDeduper{window: window, seen: make(map[string]time.Time)}
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
