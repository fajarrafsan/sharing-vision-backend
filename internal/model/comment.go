package model

import "time"

type Comment struct {
	ID         int64
	ArticleID  int64
	UserID     int64
	AuthorName string
	Body       string
	// HiddenAt terisi bila komentar disembunyikan dari publik, karena banyak
	// dilaporkan atau oleh admin.
	HiddenAt  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

const (
	ReportSpam    = "spam"
	ReportAbusive = "abusive"
	ReportOther   = "other"
)

func ValidReportReason(reason string) bool {
	return reason == ReportSpam || reason == ReportAbusive || reason == ReportOther
}

// ReportedComment adalah komentar di antrean moderasi.
type ReportedComment struct {
	Comment
	ArticleTitle   string
	ArticleSlug    string
	Reports        int
	ReasonCounts   map[string]int
	LastReportedAt *time.Time
}
