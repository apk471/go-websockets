package handler

import (
	"github.com/ayush-amin/go-boilerplate/internal/server"
	"github.com/ayush-amin/go-boilerplate/internal/service"
)

type Handlers struct {
	Health     *HealthHandler
	Match      *MatchHandler
	Commentary *CommentaryHandler
	OpenAPI    *OpenAPIHandler
	WebSocket  *WebSocketHandler
}

func NewHandlers(s *server.Server, services *service.Services) *Handlers {
	websocketHandler := NewWebSocketHandler(s)

	return &Handlers{
		Health:     NewHealthHandler(s),
		Match:      NewMatchHandler(s, services, websocketHandler),
		Commentary: NewCommentaryHandler(s, services, websocketHandler),
		OpenAPI:    NewOpenAPIHandler(s),
		WebSocket:  websocketHandler,
	}
}
