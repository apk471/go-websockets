package repository

import "github.com/apk471/go-boilerplate/internal/server"

type Repositories struct {
	Match      *MatchRepository
	Commentary *CommentaryRepository
}

func NewRepositories(s *server.Server) *Repositories {
	return &Repositories{
		Match:      NewMatchRepository(s),
		Commentary: NewCommentaryRepository(s),
	}
}
