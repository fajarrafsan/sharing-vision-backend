package pagination

import (
	"net/url"
	"strconv"

	"warta/internal/apperr"
)

// maxPage mencegah offset meluap saat dikalikan per_page.
const maxPage = 1_000_000

type Params struct {
	Page    int
	PerPage int
}

func (p Params) Limit() int {
	return p.PerPage
}

func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func NewMeta(p Params, total int64) Meta {
	pages := int((total + int64(p.PerPage) - 1) / int64(p.PerPage))
	return Meta{Page: p.Page, PerPage: p.PerPage, Total: total, TotalPages: pages}
}

type Parser struct {
	DefaultPerPage int
	MaxPerPage     int
}

// Parse membaca page dan per_page dari query string. Nilai yang bukan angka
// ditolak, sedangkan per_page yang terlalu besar dipangkas ke batas maksimum.
func (ps Parser) Parse(q url.Values) (Params, error) {
	p := Params{Page: 1, PerPage: ps.DefaultPerPage}

	if raw := q.Get("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page < 1 || page > maxPage {
			return Params{}, apperr.BadRequest("page harus berupa angka bulat positif")
		}
		p.Page = page
	}

	if raw := q.Get("per_page"); raw != "" {
		perPage, err := strconv.Atoi(raw)
		if err != nil || perPage < 1 {
			return Params{}, apperr.BadRequest("per_page harus berupa angka bulat positif")
		}
		p.PerPage = min(perPage, ps.MaxPerPage)
	}

	return p, nil
}
