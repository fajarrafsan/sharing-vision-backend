package slug

import (
	"strings"
	"testing"
)

func TestMake(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Panduan Membangun REST API dengan Golang", "panduan-membangun-rest-api-dengan-golang"},
		{"  --Halo,   Dunia!!  ", "halo-dunia"},
		{"Café Crème à la Française", "cafe-creme-a-la-francaise"},
		{"C++ & Go: 2024", "c-go-2024"},
		{"2024", "artikel-2024"},
		{"日本語", "artikel"},
		{"", "artikel"},
	}

	for _, tt := range tests {
		if got := Make(tt.in, "artikel", 200); got != tt.want {
			t.Errorf("Make(%q) = %q, ingin %q", tt.in, got, tt.want)
		}
	}
}

func TestMakeTruncatesAtWordBoundary(t *testing.T) {
	got := Make(strings.Repeat("kata ", 50), "artikel", 32)
	if len(got) > 32 || strings.HasSuffix(got, "-") || strings.HasSuffix(got, "kat") {
		t.Fatalf("pemotongan: %q", got)
	}
}

func TestUnique(t *testing.T) {
	if got := Unique("halo", nil); got != "halo" {
		t.Fatalf("slug bebas: %q", got)
	}
	if got := Unique("halo", []string{"halo", "halo-2", "halo-4"}); got != "halo-3" {
		t.Fatalf("slug terpakai: %q", got)
	}
}

func TestIsNumeric(t *testing.T) {
	for in, want := range map[string]bool{"123": true, "": false, "12a": false, "artikel-1": false} {
		if got := IsNumeric(in); got != want {
			t.Errorf("IsNumeric(%q) = %v", in, got)
		}
	}
}
