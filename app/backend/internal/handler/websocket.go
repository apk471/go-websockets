package handler

import (
	"net/http"
	"sync"

	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type wsClient struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

type WebSocketHandler struct {
	server    *server.Server
	upgrader  websocket.Upgrader
	clients   map[*websocket.Conn]*wsClient
	clientsMu sync.RWMutex
}

func NewWebSocketHandler(s *server.Server) *WebSocketHandler {
	return &WebSocketHandler{
		server:  s,
		clients: make(map[*websocket.Conn]*wsClient),
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
	client := &wsClient{conn: conn}
	h.addClient(client)
	defer h.removeClient(conn)

	h.server.Logger.Info().Msg("websocket client connected")

	for {
		_, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			if websocket.IsCloseError(
				readErr,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
			) {
				h.server.Logger.Info().Err(readErr).Msg("websocket client disconnected")
			} else {
				h.server.Logger.Error().Err(readErr).Msg("websocket read failed")
			}
			break
		}

		h.server.Logger.Info().
			Str("payload", string(payload)).
			Msg("received websocket message")

		h.broadcast("received message from client: " + string(payload))
	}

	return nil
}

func (h *WebSocketHandler) addClient(client *wsClient) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	h.clients[client.conn] = client
}

func (h *WebSocketHandler) removeClient(conn *websocket.Conn) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	delete(h.clients, conn)
}

func (h *WebSocketHandler) broadcast(message string) {
	h.clientsMu.RLock()
	clients := make([]*wsClient, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.clientsMu.RUnlock()

	for _, client := range clients {
		client.writeMu.Lock()
		err := client.conn.WriteMessage(websocket.TextMessage, []byte(message))
		client.writeMu.Unlock()
		if err != nil {
			h.server.Logger.Error().Err(err).Msg("websocket broadcast write failed")
			h.removeClient(client.conn)
			_ = client.conn.Close()
		}
	}
}
