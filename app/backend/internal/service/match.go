package service

import (
	"context"
	"strings"
	"time"

	"github.com/apk471/go-boilerplate/internal/model"
	"github.com/apk471/go-boilerplate/internal/repository"
)

type CreateMatchInput struct {
	Sport     string
	HomeTeam  string
	AwayTeam  string
	StartTime time.Time
	EndTime   time.Time
}

type MatchService struct {
	repos *repository.Repositories
}

func NewMatchService(repos *repository.Repositories) *MatchService {
	return &MatchService{
		repos: repos,
	}
}

func (s *MatchService) CreateMatch(ctx context.Context, input CreateMatchInput) (model.Match, error) {
	return s.repos.Match.CreateMatch(ctx, repository.CreateMatchParams{
		Sport:     strings.TrimSpace(input.Sport),
		HomeTeam:  strings.TrimSpace(input.HomeTeam),
		AwayTeam:  strings.TrimSpace(input.AwayTeam),
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
	})
}

func (s *MatchService) ListMatches(ctx context.Context, limit int) ([]model.Match, error) {
	if limit <= 0 {
		limit = 50
	}

	return s.repos.Match.ListMatches(ctx, limit)
}
