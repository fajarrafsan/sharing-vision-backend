package model

import (
	"regexp"
	"strings"
)

var (
	mdImage    = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	mdLink     = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	mdBlock    = regexp.MustCompile(`^\s{0,3}(#{1,6}\s+|>\s?|[-*+]\s+|\d+[.)]\s+)`)
	mdEmphasis = regexp.MustCompile("(\\*\\*|__|~~|\\*|`)")
	mdRule     = regexp.MustCompile(`^\s{0,3}([-*_]\s*){3,}$`)
	mdHeading  = regexp.MustCompile(`^\s{0,3}#{1,6}\s+`)
)

// PlainText membuang sintaks Markdown yang umum (judul, kutipan, daftar,
// tautan, gambar, penebalan, blok kode) sehingga isi bisa dipakai sebagai
// cuplikan. Ini bukan parser Markdown lengkap, cukup untuk teks ringkasan.
func PlainText(markdown string) string {
	var out []string
	inFence := false

	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence || mdRule.MatchString(line) {
			continue
		}

		heading := mdHeading.MatchString(line)
		line = mdBlock.ReplaceAllString(line, "")
		line = mdImage.ReplaceAllString(line, "$1")
		line = mdLink.ReplaceAllString(line, "$1")
		line = mdEmphasis.ReplaceAllString(line, "")
		// Subjudul diberi titik supaya tidak menyambung ke kalimat berikutnya
		// di cuplikan.
		if heading && !strings.ContainsAny(line[max(0, len(line)-1):], ".!?:") {
			line = strings.TrimRight(line, " ") + "."
		}
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}
