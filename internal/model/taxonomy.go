package model

import "time"

type Category struct {
	ID          int64
	Name        string
	Slug        string
	Description string
	// ArticleCount hanya menghitung article yang sudah terbit.
	ArticleCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Tag struct {
	ID   int64
	Name string
	Slug string
	// ArticleCount hanya menghitung article yang sudah terbit.
	ArticleCount int
	CreatedAt    time.Time
}
