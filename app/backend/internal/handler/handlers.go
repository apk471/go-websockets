package handler

import (
	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/apk471/go-boilerplate/internal/service"
)

type Handlers struct {
	Health    *HealthHandler
	Match     *MatchHandler
	OpenAPI   *OpenAPIHandler
	WebSocket *WebSocketHandler
}

func NewHandlers(s *server.Server, services *service.Services) *Handlers {
	return &Handlers{
		Health:    NewHealthHandler(s),
		Match:     NewMatchHandler(s, services),
		OpenAPI:   NewOpenAPIHandler(s),
		WebSocket: NewWebSocketHandler(s),
	}
}
