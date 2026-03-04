package handler

import (
	"net/http"

	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type WebSocketHandler struct {
	server   *server.Server
	upgrader websocket.Upgrader
}

func NewWebSocketHandler(s *server.Server) *WebSocketHandler {
	return &WebSocketHandler{
		server: s,
		upgrader: websocket.Upgrader{
			// Basic setup for local/frontend testing.
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *WebSocketHandler) Handle(c echo.Context) error {
	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		h.server.Logger.Error().Err(err).Msg("failed to upgrade websocket connection")
		return err
	}
	defer conn.Close()

	h.server.Logger.Info().Msg("websocket client connected")

	for {
		messageType, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			if websocket.IsCloseError(
				readErr,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
			) {
				h.server.Logger.Error().Err(readErr).Msg("websocket client disconnected")
			} else {
				h.server.Logger.Error().Err(readErr).Msg("websocket read failed")
			}
			break
		}

		h.server.Logger.Info().
			Int("message_type", messageType).
			Str("payload", string(payload)).
			Msg("received websocket message")
	}

	return nil
}
