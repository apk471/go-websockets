package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ayush-amin/go-boilerplate/internal/model"
	"github.com/ayush-amin/go-boilerplate/internal/server"
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

func (r *MatchRepository) ListMatches(ctx context.Context, limit int) ([]model.Match, error) {
	const query = `
		SELECT id, sport, home_team, away_team, status, start_time, end_time, home_score, away_score, created_at
		FROM matches
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.server.DB.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("listing matches: %w", err)
	}
	defer rows.Close()

	matches := make([]model.Match, 0, limit)
	for rows.Next() {
		var match model.Match
		if scanErr := rows.Scan(
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
		); scanErr != nil {
			return nil, fmt.Errorf("scanning match row: %w", scanErr)
		}
		matches = append(matches, match)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating match rows: %w", err)
	}

	return matches, nil
}
