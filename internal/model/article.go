package model

import (
	"strings"
	"time"
)

type ArticleStatus string

const (
	StatusDraft     ArticleStatus = "draft"
	StatusPublished ArticleStatus = "published"
	StatusArchived  ArticleStatus = "archived"
)

func (s ArticleStatus) Valid() bool {
	switch s {
	case StatusDraft, StatusPublished, StatusArchived:
		return true
	}
	return false
}

// Article memuat kolom tabel articles beserta data hasil join yang selalu
// ikut ditampilkan: nama penulis, kategori, tag, dan jumlah komentar.
type Article struct {
	ID           int64
	Title        string
	Slug         string
	Content      string
	Excerpt      string
	Status       ArticleStatus
	AuthorID     int64
	AuthorName   string
	CategoryID   int64
	CategoryName string
	CategorySlug string
	Tags         []Tag
	CommentCount int
	PublishedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (a Article) IsPublished() bool {
	return a.Status == StatusPublished
}

func (a Article) TagNames() []string {
	names := make([]string, 0, len(a.Tags))
	for _, t := range a.Tags {
		names = append(names, t.Name)
	}
	return names
}

// Excerpt meringkas teks menjadi paling banyak n karakter, dipotong di batas
// kata supaya tidak berhenti di tengah kata.
func Excerpt(text string, n int) string {
	text = strings.Join(strings.Fields(text), " ")

	runes := []rune(text)
	if len(runes) <= n {
		return text
	}

	cut := string(runes[:n])
	// Bila karakter berikutnya spasi, potongan sudah berhenti di akhir kata.
	if runes[n] != ' ' {
		if i := strings.LastIndex(cut, " "); i > 0 {
			cut = cut[:i]
		}
	}
	return strings.TrimRight(cut, " ,.;:-") + "…"
}
