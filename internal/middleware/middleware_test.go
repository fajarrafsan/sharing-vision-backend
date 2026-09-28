package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"warta/internal/auth"
	"warta/internal/model"
)

func TestRateLimiter(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewRateLimiter(3)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("permintaan ke-%d seharusnya lolos", i+1)
		}
	}

	ok, wait := l.Allow("a")
	if ok || wait != 20*time.Second {
		t.Fatalf("seharusnya ditolak dengan jeda 20 detik: %v %v", ok, wait)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("kunci lain punya jatah sendiri")
	}

	now = now.Add(20 * time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("token seharusnya terisi lagi")
	}

	now = now.Add(2 * time.Minute)
	l.Allow("c")
	if _, found := l.buckets["a"]; found {
		t.Fatal("bucket yang sudah penuh seharusnya dibuang")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	h := RateLimit(NewRateLimiter(1))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req.RemoteAddr = "10.0.0.1:5000"

	first := httptest.NewRecorder()
	h.ServeHTTP(first, req)
	second := httptest.NewRecorder()
	h.ServeHTTP(second, req)

	if first.Code != http.StatusOK || second.Code != http.StatusTooManyRequests || second.Header().Get("Retry-After") == "" {
		t.Fatalf("status %d lalu %d, Retry-After %q", first.Code, second.Code, second.Header().Get("Retry-After"))
	}
}

func TestAuthenticate(t *testing.T) {
	tokens := auth.NewTokenManager("rahasia-test-yang-panjangnya-lebih-dari-32", "warta", time.Minute)
	author, _ := tokens.Issue(model.User{ID: 5, Role: model.RoleAuthor})

	var seen auth.Actor
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = auth.ActorFrom(r.Context())
	})

	tests := []struct {
		name   string
		header string
		chain  []Middleware
		status int
		actor  int64
	}{
		{"anonim di rute publik", "", nil, http.StatusOK, 0},
		{"token sah", "Bearer " + author, nil, http.StatusOK, 5},
		{"skema huruf kecil", "bearer " + author, nil, http.StatusOK, 5},
		{"token rusak", "Bearer abc", nil, http.StatusUnauthorized, 0},
		{"tanpa skema", author, nil, http.StatusUnauthorized, 0},
		{"anonim di rute wajib login", "", []Middleware{RequireAuth}, http.StatusUnauthorized, 0},
		{"author di rute admin", "Bearer " + author, []Middleware{RequireAuth, RequireRole(model.RoleAdmin)}, http.StatusForbidden, 0},
		{"author di rute penulis", "Bearer " + author, []Middleware{RequireAuth, RequireRole(model.RoleAdmin, model.RoleAuthor)}, http.StatusOK, 5},
	}

	for _, tt := range tests {
		seen = auth.Actor{}
		h := Chain(inner, append([]Middleware{Authenticate(tokens)}, tt.chain...)...)

		req := httptest.NewRequest("GET", "/", nil)
		if tt.header != "" {
			req.Header.Set("Authorization", tt.header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != tt.status || seen.ID != tt.actor {
			t.Errorf("%s: status %d actor %d", tt.name, rec.Code, seen.ID)
		}
	}
}

func TestCORS(t *testing.T) {
	h := CORS([]string{"https://warta.id"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	preflight := httptest.NewRequest("OPTIONS", "/api/v1/articles", nil)
	preflight.Header.Set("Origin", "https://warta.id")
	preflight.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, preflight)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "https://warta.id" {
		t.Fatalf("preflight origin terdaftar: %d %v", rec.Code, rec.Header())
	}

	other := httptest.NewRequest("GET", "/api/v1/articles", nil)
	other.Header.Set("Origin", "https://jahat.example")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, other)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("origin yang tidak terdaftar tidak boleh diizinkan")
	}
}

func TestRecover(t *testing.T) {
	h := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
}
