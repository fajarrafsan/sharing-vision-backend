package service

import (
	"context"
	"time"

	"warta/internal/apperr"
	"warta/internal/auth"
	"warta/internal/dto"
	"warta/internal/repository"
)

const (
	MinStatsDays     = 7
	MaxStatsDays     = 90
	DefaultStatsDays = 30
)

type StatsService interface {
	// Overview meringkas aktivitas days hari terakhir. Admin melihat semua
	// artikel, author hanya artikelnya sendiri.
	Overview(ctx context.Context, actor auth.Actor, days int) (dto.StatsResponse, error)
}

type statsService struct {
	stats repository.StatsRepository
	now   func() time.Time
}

func NewStatsService(stats repository.StatsRepository) StatsService {
	return &statsService{stats: stats, now: time.Now}
}

func (s *statsService) Overview(ctx context.Context, actor auth.Actor, days int) (dto.StatsResponse, error) {
	if days < MinStatsDays || days > MaxStatsDays {
		return dto.StatsResponse{}, apperr.BadRequest("days harus antara 7 dan 90")
	}

	response := dto.StatsResponse{Scope: "all", Days: days}
	var authorID int64
	if !actor.IsAdmin() {
		response.Scope = "mine"
		authorID = actor.ID
	}

	totals, err := s.stats.Totals(ctx, authorID)
	if err != nil {
		return dto.StatsResponse{}, apperr.Internal(err)
	}
	response.Totals = dto.StatsTotals(totals)

	today := s.now().UTC().Truncate(24 * time.Hour)
	from := today.AddDate(0, 0, -(days - 1))
	activity, err := s.stats.Daily(ctx, authorID, from)
	if err != nil {
		return dto.StatsResponse{}, apperr.Internal(err)
	}
	response.Daily = fillDays(from, days, activity)

	top, err := s.stats.TopArticles(ctx, authorID, 5)
	if err != nil {
		return dto.StatsResponse{}, apperr.Internal(err)
	}
	response.TopArticles = make([]dto.TopArticle, 0, len(top))
	for _, t := range top {
		response.TopArticles = append(response.TopArticles, dto.TopArticle(t))
	}

	if actor.IsAdmin() {
		users, err := s.stats.UsersByRole(ctx)
		if err != nil {
			return dto.StatsResponse{}, apperr.Internal(err)
		}
		response.Users = make(map[string]int64, len(users))
		for role, n := range users {
			response.Users[string(role)] = n
		}
	}

	return response, nil
}

// fillDays membuat satu baris per hari sejak from, termasuk hari tanpa
// aktivitas, supaya grafik tidak melompati tanggal.
func fillDays(from time.Time, days int, activity []repository.DailyActivity) []dto.DailyStat {
	byDate := make(map[string]repository.DailyActivity, len(activity))
	for _, a := range activity {
		byDate[a.Day.Format(time.DateOnly)] = a
	}

	daily := make([]dto.DailyStat, 0, days)
	for i := 0; i < days; i++ {
		date := from.AddDate(0, 0, i).Format(time.DateOnly)
		a := byDate[date]
		daily = append(daily, dto.DailyStat{Date: date, Views: a.Views, Comments: a.Comments, Published: a.Published})
	}
	return daily
}
