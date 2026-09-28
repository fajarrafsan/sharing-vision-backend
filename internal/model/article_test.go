package model

import "testing"

func TestExcerpt(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"pendek saja", 50, "pendek saja"},
		{"  spasi   berlebih\ndan baris  baru ", 50, "spasi berlebih dan baris baru"},
		{"satu dua tiga empat lima", 12, "satu dua…"},
		{"kalimat berakhir koma, lalu lanjut", 22, "kalimat berakhir koma…"},
		{"ééééé ééééé", 7, "ééééé…"},
	}

	for _, tt := range tests {
		if got := Excerpt(tt.in, tt.n); got != tt.want {
			t.Errorf("Excerpt(%q, %d) = %q, ingin %q", tt.in, tt.n, got, tt.want)
		}
	}
}
