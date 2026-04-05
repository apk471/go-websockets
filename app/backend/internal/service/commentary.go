package service

import (
	"context"
	"strings"

	"github.com/apk471/go-boilerplate/internal/model"
	"github.com/apk471/go-boilerplate/internal/repository"
)

const maxCommentaryLimit = 100

type CreateCommentaryInput struct {
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

type CommentaryService struct {
	repos *repository.Repositories
}

func NewCommentaryService(repos *repository.Repositories) *CommentaryService {
	return &CommentaryService{
		repos: repos,
	}
}

func (s *CommentaryService) ListCommentary(ctx context.Context, matchID, limit int) ([]model.Commentary, error) {
	if limit <= 0 || limit > maxCommentaryLimit {
		limit = maxCommentaryLimit
	}

	return s.repos.Commentary.ListCommentary(ctx, matchID, limit)
}

func (s *CommentaryService) CreateCommentary(ctx context.Context, input CreateCommentaryInput) (model.Commentary, error) {
	return s.repos.Commentary.CreateCommentary(ctx, repository.CreateCommentaryParams{
		MatchID:   input.MatchID,
		Minute:    input.Minute,
		Sequence:  input.Sequence,
		Period:    trimOptionalString(input.Period),
		EventType: trimOptionalString(input.EventType),
		Actor:     trimOptionalString(input.Actor),
		Team:      trimOptionalString(input.Team),
		Message:   strings.TrimSpace(input.Message),
		Metadata:  input.Metadata,
		Tags:      trimTags(input.Tags),
	})
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func trimTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	trimmed := make([]string, 0, len(tags))
	for _, tag := range tags {
		cleaned := strings.TrimSpace(tag)
		if cleaned == "" {
			continue
		}
		trimmed = append(trimmed, cleaned)
	}

	if len(trimmed) == 0 {
		return nil
	}

	return trimmed
}
