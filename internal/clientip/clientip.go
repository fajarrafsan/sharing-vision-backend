// Package clientip menentukan alamat IP asli pengunjung.
//
// Di balik reverse proxy (Nginx, load balancer), koneksi ke service datang
// dari proxy itu, sehingga semua pengunjung terlihat berasal dari satu IP.
// Proxy menambahkan IP asli ke header X-Forwarded-For. Header itu bisa diisi
// sembarangan oleh klien, jadi hanya dipercaya bila koneksi datang dari proxy
// yang terdaftar di TRUSTED_PROXIES.
package clientip

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type Resolver struct {
	trusted []netip.Prefix
}

// NewResolver menerima daftar IP atau CIDR proxy yang dipercaya, misalnya
// "10.0.0.0/8" atau "127.0.0.1". Daftar kosong berarti tidak ada proxy:
// alamat koneksi dipakai apa adanya.
func NewResolver(proxies []string) (*Resolver, error) {
	r := &Resolver{}
	for _, raw := range proxies {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if !strings.Contains(raw, "/") {
			addr, err := netip.ParseAddr(raw)
			if err != nil {
				return nil, fmt.Errorf("proxy %q bukan IP atau CIDR yang sah", raw)
			}
			r.trusted = append(r.trusted, netip.PrefixFrom(addr.Unmap(), addr.Unmap().BitLen()))
			continue
		}
		prefix, err := netip.ParsePrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("proxy %q bukan IP atau CIDR yang sah", raw)
		}
		r.trusted = append(r.trusted, prefix.Masked())
	}
	return r, nil
}

func (r *Resolver) isTrusted(addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, p := range r.trusted {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// IP mengembalikan IP pengunjung. X-Forwarded-For dibaca dari kanan ke kiri:
// entri paling kanan ditambahkan proxy terdekat, jadi entri pertama yang
// bukan proxy tepercaya adalah pengunjungnya. Entri di sebelah kirinya bisa
// dipalsukan klien dan diabaikan.
func (r *Resolver) IP(req *http.Request) string {
	remote := hostOnly(req.RemoteAddr)
	addr, err := netip.ParseAddr(remote)
	if err != nil || len(r.trusted) == 0 || !r.isTrusted(addr) {
		return remote
	}

	var hops []string
	for _, header := range req.Header.Values("X-Forwarded-For") {
		hops = append(hops, strings.Split(header, ",")...)
	}

	for i := len(hops) - 1; i >= 0; i-- {
		hop, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			// Isi yang rusak tidak bisa dipercaya; berhenti di proxy terakhir.
			return addr.Unmap().String()
		}
		if !r.isTrusted(hop) {
			return hop.Unmap().String()
		}
		addr = hop
	}
	return addr.Unmap().String()
}

func hostOnly(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

type ctxKey struct{}

// Middleware menaruh IP pengunjung di context untuk dibaca lapisan lain.
func (r *Resolver) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), ctxKey{}, r.IP(req))))
	})
}

// From mengembalikan IP dari context, atau alamat koneksi bila middleware
// tidak dipasang.
func From(req *http.Request) string {
	if ip, ok := req.Context().Value(ctxKey{}).(string); ok {
		return ip
	}
	return hostOnly(req.RemoteAddr)
}
