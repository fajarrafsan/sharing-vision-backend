package slug

import (
	"strconv"
	"strings"
)

var latin = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'æ': "ae",
	'ç': "c", 'è': "e", 'é': "e", 'ê': "e", 'ë': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ñ': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'œ': "oe",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ý': "y", 'ÿ': "y", 'ß': "ss",
}

// Make mengubah teks menjadi slug yang hanya berisi a-z, 0-9, dan tanda
// hubung, paling panjang maxLen karakter.
//
// Slug yang kosong diganti fallback, dan slug yang seluruhnya angka diberi
// awalan fallback. Endpoint menerima id maupun slug di posisi yang sama, jadi
// slug tidak boleh bisa dikira id.
func Make(text, fallback string, maxLen int) string {
	var b strings.Builder
	dash := false

	for _, r := range strings.ToLower(text) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case latin[r] != "":
			b.WriteString(latin[r])
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}

	s := strings.TrimRight(b.String(), "-")

	if len(s) > maxLen {
		s = s[:maxLen]
		if i := strings.LastIndexByte(s, '-'); i > maxLen/2 {
			s = s[:i]
		}
		s = strings.TrimRight(s, "-")
	}

	if s == "" {
		return fallback
	}
	if IsNumeric(s) {
		return fallback + "-" + s
	}
	return s
}

// Unique mengembalikan base bila belum dipakai, atau base-2, base-3, dan
// seterusnya.
func Unique(base string, taken []string) string {
	used := make(map[string]bool, len(taken))
	for _, t := range taken {
		used[t] = true
	}

	if !used[base] {
		return base
	}
	for n := 2; ; n++ {
		candidate := base + "-" + strconv.Itoa(n)
		if !used[candidate] {
			return candidate
		}
	}
}

func IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
