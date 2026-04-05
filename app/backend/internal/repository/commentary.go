package repository

import (
	"context"
	"fmt"

	"github.com/apk471/go-boilerplate/internal/model"
	"github.com/apk471/go-boilerplate/internal/server"
)

type CreateCommentaryParams struct {
	MatchID   int
	Minute    int
	Sequence  *int
	Period    *string
	EventType *string
	Actor     *string
	Team      *string
	Message   string
	Metadata  map[string]any
	Tags      []string
}

type CommentaryRepository struct {
	server *server.Server
}

func NewCommentaryRepository(s *server.Server) *CommentaryRepository {
	return &CommentaryRepository{
		server: s,
	}
}

func (r *CommentaryRepository) ListCommentary(ctx context.Context, matchID, limit int) ([]model.Commentary, error) {
	const query = `
		SELECT
			id,
			match_id,
			minute,
			sequence,
			period,
			event_type,
			actor,
			team,
			message,
			COALESCE(metadata, '{}'::jsonb),
			COALESCE(tags, '{}'::text[]),
			created_at
		FROM commentary
		WHERE match_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.server.DB.Pool.Query(ctx, query, matchID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing commentary: %w", err)
	}
	defer rows.Close()

	commentary := make([]model.Commentary, 0, limit)
	for rows.Next() {
		var item model.Commentary
		if scanErr := rows.Scan(
			&item.ID,
			&item.MatchID,
			&item.Minute,
			&item.Sequence,
			&item.Period,
			&item.EventType,
			&item.Actor,
			&item.Team,
			&item.Message,
			&item.Metadata,
			&item.Tags,
			&item.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scanning commentary row: %w", scanErr)
		}
		commentary = append(commentary, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating commentary rows: %w", err)
	}

	return commentary, nil
}

func (r *CommentaryRepository) CreateCommentary(ctx context.Context, params CreateCommentaryParams) (model.Commentary, error) {
	const query = `
		INSERT INTO commentary (
			match_id,
			minute,
			sequence,
			period,
			event_type,
			actor,
			team,
			message,
			metadata,
			tags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING
			id,
			match_id,
			minute,
			sequence,
			period,
			event_type,
			actor,
			team,
			message,
			COALESCE(metadata, '{}'::jsonb),
			COALESCE(tags, '{}'::text[]),
			created_at
	`

	var item model.Commentary
	err := r.server.DB.Pool.QueryRow(
		ctx,
		query,
		params.MatchID,
		params.Minute,
		params.Sequence,
		params.Period,
		params.EventType,
		params.Actor,
		params.Team,
		params.Message,
		params.Metadata,
		params.Tags,
	).Scan(
		&item.ID,
		&item.MatchID,
		&item.Minute,
		&item.Sequence,
		&item.Period,
		&item.EventType,
		&item.Actor,
		&item.Team,
		&item.Message,
		&item.Metadata,
		&item.Tags,
		&item.CreatedAt,
	)
	if err != nil {
		return model.Commentary{}, fmt.Errorf("creating commentary: %w", err)
	}

	return item, nil
}
