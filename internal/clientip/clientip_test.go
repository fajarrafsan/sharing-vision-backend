package clientip

import (
	"net/http/httptest"
	"testing"
)

func TestIP(t *testing.T) {
	r, err := NewResolver([]string{"10.0.0.0/8", "192.168.1.5"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		remote string
		xff    []string
		want   string
	}{
		{"tanpa proxy", "203.0.113.7:5000", nil, "203.0.113.7"},
		{"klien langsung memalsukan header", "203.0.113.7:5000", []string{"1.1.1.1"}, "203.0.113.7"},
		{"lewat satu proxy", "10.0.0.2:80", []string{"198.51.100.9"}, "198.51.100.9"},
		{"klien menambah entri palsu di kiri", "10.0.0.2:80", []string{"1.1.1.1, 198.51.100.9"}, "198.51.100.9"},
		{"lewat dua proxy", "10.0.0.2:80", []string{"198.51.100.9, 192.168.1.5"}, "198.51.100.9"},
		{"header ganda", "10.0.0.2:80", []string{"1.1.1.1", "198.51.100.9"}, "198.51.100.9"},
		{"proxy tanpa header", "10.0.0.2:80", nil, "10.0.0.2"},
		{"entri rusak", "10.0.0.2:80", []string{"bukan-ip"}, "10.0.0.2"},
		{"IPv6", "[::1]:80", []string{"198.51.100.9"}, "::1"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = tt.remote
		for _, v := range tt.xff {
			req.Header.Add("X-Forwarded-For", v)
		}
		if got := r.IP(req); got != tt.want {
			t.Errorf("%s: %s, ingin %s", tt.name, got, tt.want)
		}
	}
}

func TestNoTrustedProxies(t *testing.T) {
	r, _ := NewResolver(nil)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.2:80"
	req.Header.Set("X-Forwarded-For", "198.51.100.9")
	if got := r.IP(req); got != "10.0.0.2" {
		t.Fatalf("tanpa TRUSTED_PROXIES header harus diabaikan: %s", got)
	}
}

func TestNewResolverRejectsGarbage(t *testing.T) {
	for _, bad := range []string{"bukan-ip", "10.0.0.0/99"} {
		if _, err := NewResolver([]string{bad}); err == nil {
			t.Errorf("%q seharusnya ditolak", bad)
		}
	}
}
