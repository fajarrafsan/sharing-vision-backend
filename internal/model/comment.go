package model

import "time"

type Comment struct {
	ID         int64
	ArticleID  int64
	UserID     int64
	AuthorName string
	Body       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
