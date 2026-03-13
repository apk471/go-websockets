package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/apk471/go-boilerplate/internal/model"
	"github.com/apk471/go-boilerplate/internal/server"
)

type CreateMatchParams struct {
	Sport     string
	HomeTeam  string
	AwayTeam  string
	StartTime time.Time
	EndTime   time.Time
}

type MatchRepository struct {
	server *server.Server
}

func NewMatchRepository(s *server.Server) *MatchRepository {
	return &MatchRepository{
		server: s,
	}
}

func (r *MatchRepository) CreateMatch(ctx context.Context, params CreateMatchParams) (model.Match, error) {
	const query = `
		INSERT INTO matches (
			sport,
			home_team,
			away_team,
			start_time,
			end_time
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, sport, home_team, away_team, status, start_time, end_time, home_score, away_score, created_at
	`

	var match model.Match
	err := r.server.DB.Pool.QueryRow(
		ctx,
		query,
		params.Sport,
		params.HomeTeam,
		params.AwayTeam,
		params.StartTime,
		params.EndTime,
	).Scan(
		&match.ID,
		&match.Sport,
		&match.HomeTeam,
		&match.AwayTeam,
		&match.Status,
		&match.StartTime,
		&match.EndTime,
		&match.HomeScore,
		&match.AwayScore,
		&match.CreatedAt,
	)
	if err != nil {
		return model.Match{}, fmt.Errorf("creating match: %w", err)
	}

	return match, nil
}
