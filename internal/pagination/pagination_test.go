package pagination

import (
	"net/url"
	"testing"
)

func TestParse(t *testing.T) {
	ps := Parser{DefaultPerPage: 10, MaxPerPage: 50}

	tests := []struct {
		query   string
		want    Params
		wantErr bool
	}{
		{"", Params{Page: 1, PerPage: 10}, false},
		{"page=3&per_page=20", Params{Page: 3, PerPage: 20}, false},
		{"per_page=500", Params{Page: 1, PerPage: 50}, false},
		{"page=0", Params{}, true},
		{"page=abc", Params{}, true},
		{"per_page=-1", Params{}, true},
		{"page=99999999999", Params{}, true},
	}

	for _, tt := range tests {
		q, _ := url.ParseQuery(tt.query)
		got, err := ps.Parse(q)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("Parse(%q) = %+v, %v", tt.query, got, err)
		}
	}
}

func TestMeta(t *testing.T) {
	p := Params{Page: 2, PerPage: 10}
	if p.Offset() != 10 {
		t.Fatalf("offset: %d", p.Offset())
	}

	for total, pages := range map[int64]int{0: 0, 1: 1, 10: 1, 11: 2} {
		if m := NewMeta(p, total); m.TotalPages != pages {
			t.Errorf("total %d: total_pages %d, ingin %d", total, m.TotalPages, pages)
		}
	}
}
