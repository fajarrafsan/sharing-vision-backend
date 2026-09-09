package model

import "time"

const (
	StatusPublish = "publish"
	StatusDraft   = "draft"
	StatusThrash  = "thrash"
)

type Article struct {
	ID          int64
	Title       string
	Content     string
	Category    string
	CreatedDate time.Time
	UpdatedDate time.Time
	Status      string
}
