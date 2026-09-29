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

func TestPlainText(t *testing.T) {
	markdown := "# Judul Besar\n\nParagraf dengan **tebal**, _miring_, dan `kode`.\n\n" +
		"- butir [tautan](https://warta.id)\n1. nomor ![gambar](/uploads/a.png)\n> kutipan\n\n" +
		"```go\nfmt.Println(\"abaikan\")\n```\n---\nPenutup ~~coret~~."

	got := Excerpt(markdown, 500)
	want := "Judul Besar Paragraf dengan tebal, _miring_, dan kode. butir tautan nomor gambar kutipan Penutup coret."
	if got != want {
		t.Fatalf("Excerpt markdown:\n got %q\nwant %q", got, want)
	}
}

func TestReadingMinutes(t *testing.T) {
	for length, want := range map[int]int{0: 1, 1200: 1, 1201: 2, 6000: 5} {
		if got := (Article{ContentLength: length}).ReadingMinutes(); got != want {
			t.Errorf("ContentLength %d: %d menit, ingin %d", length, got, want)
		}
	}
}
