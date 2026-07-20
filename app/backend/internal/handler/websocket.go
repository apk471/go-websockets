package handler

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ayush-amin/go-boilerplate/internal/errs"
	appMiddleware "github.com/ayush-amin/go-boilerplate/internal/middleware"
	"github.com/ayush-amin/go-boilerplate/internal/model"
	"github.com/ayush-amin/go-boilerplate/internal/server"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type wsClient struct {
	conn          *websocket.Conn
	ip            string
	writeMu       sync.Mutex
	subscriptions map[int]struct{}
}

type websocketMessage struct {
	Type    string `json:"type"`
	MatchID *int   `json:"matchId,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type websocketClientMessage struct {
	Type    string `json:"type"`
	MatchID *int   `json:"matchId,omitempty"`
}

type WebSocketHandler struct {
	server                *server.Server
	rateLimit             *appMiddleware.RateLimitMiddleware
	upgrader              websocket.Upgrader
	clients               map[*websocket.Conn]*wsClient
	commentarySubscribers map[int]map[*websocket.Conn]*wsClient
	clientsByIP           map[string]int
	clientsMu             sync.RWMutex
	pingInterval          time.Duration
	pongWait              time.Duration
	writeTimeout          time.Duration
	maxMessageSize        int64
}

func NewWebSocketHandler(s *server.Server) *WebSocketHandler {
	handler := &WebSocketHandler{
		server:                s,
		rateLimit:             appMiddleware.NewRateLimitMiddleware(s),
		clients:               make(map[*websocket.Conn]*wsClient),
		commentarySubscribers: make(map[int]map[*websocket.Conn]*wsClient),
		clientsByIP:           make(map[string]int),
		pingInterval:          time.Duration(s.Config.Server.WSPingInterval) * time.Second,
		pongWait:              time.Duration(s.Config.Server.WSPongWait) * time.Second,
		writeTimeout:          time.Duration(s.Config.Server.WSWriteTimeout) * time.Second,
		maxMessageSize:        s.Config.Server.WSMaxMessageBytes,
	}

	handler.upgrader = websocket.Upgrader{
		HandshakeTimeout: time.Duration(s.Config.Server.WSHandshakeTimeout) * time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		CheckOrigin:      handler.checkOrigin,
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			http.Error(w, http.StatusText(status), status)
		},
	}

	return handler
}

func (h *WebSocketHandler) Handle(c echo.Context) error {
	ip := c.RealIP()
	if !h.rateLimit.AllowWSUpgrade(ip) {
		h.rateLimit.RecordRateLimitHit("websocket_upgrade", c.Path(), ip)
		return errs.NewTooManyRequestsError("Too many websocket upgrade attempts", false)
	}

	if !h.allowConnection(ip) {
		h.server.Logger.Warn().
			Str("ip", ip).
			Int("max_connections_per_ip", h.server.Config.Server.WSMaxConnectionsIP).
			Msg("websocket connection denied because connection limit was reached")
		return errs.NewTooManyRequestsError("Too many active websocket connections", false)
	}

	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		h.releaseConnection(ip)
		h.server.Logger.Error().Err(err).Msg("failed to upgrade websocket connection")
		return err
	}
	defer conn.Close()

	client := &wsClient{
		conn:          conn,
		ip:            wsIdentifier(ip),
		subscriptions: make(map[int]struct{}),
	}
	h.addClient(client)
	defer h.removeClient(conn, client.ip)

	h.configureConnection(conn)

	done := make(chan struct{})
	go h.writePump(client, done)
	defer close(done)

	h.server.Logger.Info().Msg("websocket client connected")

	if err := h.writeJSON(client, websocketMessage{
		Type:    "welcome",
		Message: "welcome",
	}); err != nil {
		h.server.Logger.Error().Err(err).Msg("failed to send websocket welcome message")
		return nil
	}

	for {
		messageType, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			if websocket.IsCloseError(
				readErr,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
			) {
				h.server.Logger.Info().Err(readErr).Msg("websocket client disconnected")
			} else if errors.Is(readErr, net.ErrClosed) {
				h.server.Logger.Info().Msg("websocket connection closed")
			} else {
				h.server.Logger.Error().Err(readErr).Msg("websocket read failed")
			}
			break
		}

		if messageType == websocket.BinaryMessage {
			_ = h.writeControl(client, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "binary messages are not supported"))
			break
		}

		if messageType != websocket.TextMessage {
			continue
		}

		if !h.rateLimit.AllowWSMessage(ip) {
			h.rateLimit.RecordRateLimitHit("websocket_message", c.Path(), ip)
			_ = h.writeControl(client, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "message rate limit exceeded"))
			break
		}

		if err := h.handleMessage(client, payload); err != nil {
			h.server.Logger.Warn().
				Err(err).
				Str("ip", ip).
				Msg("websocket message handling failed")

			if writeErr := h.writeJSON(client, websocketMessage{
				Type:    "error",
				Message: err.Error(),
			}); writeErr != nil {
				h.server.Logger.Warn().Err(writeErr).Msg("failed to send websocket error message")
				break
			}
		}
	}

	return nil
}

func (h *WebSocketHandler) BroadcastMatchCreated(match model.Match) {
	h.Broadcast(websocketMessage{
		Type: "match.created",
		Data: match,
	})
}

func (h *WebSocketHandler) BroadcastCommentaryCreated(commentary model.Commentary) {
	h.broadcastToCommentarySubscribers(commentary.MatchID, websocketMessage{
		Type: "commentary.created",
		Data: commentary,
	})
}

func (h *WebSocketHandler) addClient(client *wsClient) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	h.clients[client.conn] = client
}

func (h *WebSocketHandler) removeClient(conn *websocket.Conn, ip string) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	client := h.clients[conn]
	delete(h.clients, conn)
	if client != nil {
		for matchID := range client.subscriptions {
			subscribers := h.commentarySubscribers[matchID]
			delete(subscribers, conn)
			if len(subscribers) == 0 {
				delete(h.commentarySubscribers, matchID)
			}
		}
	}
	identifier := wsIdentifier(ip)
	if h.clientsByIP[identifier] > 0 {
		h.clientsByIP[identifier]--
		if h.clientsByIP[identifier] == 0 {
			delete(h.clientsByIP, identifier)
		}
	}
}

func (h *WebSocketHandler) Broadcast(message websocketMessage) {
	h.clientsMu.RLock()
	clients := make([]*wsClient, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.clientsMu.RUnlock()

	for _, client := range clients {
		err := h.writeJSON(client, message)
		if err != nil {
			h.server.Logger.Error().Err(err).Msg("websocket broadcast write failed")
			h.removeClient(client.conn, client.ip)
			_ = client.conn.Close()
		}
	}
}

func (h *WebSocketHandler) broadcastToCommentarySubscribers(matchID int, message websocketMessage) {
	h.clientsMu.RLock()
	subscribers := h.commentarySubscribers[matchID]
	clients := make([]*wsClient, 0, len(subscribers))
	for _, client := range subscribers {
		clients = append(clients, client)
	}
	h.clientsMu.RUnlock()

	for _, client := range clients {
		if err := h.writeJSON(client, message); err != nil {
			h.server.Logger.Error().Err(err).Int("match_id", matchID).Msg("websocket commentary broadcast write failed")
			h.removeClient(client.conn, client.ip)
			_ = client.conn.Close()
		}
	}
}

func (h *WebSocketHandler) writeJSON(client *wsClient, message websocketMessage) error {
	client.writeMu.Lock()
	defer client.writeMu.Unlock()

	if err := client.conn.SetWriteDeadline(time.Now().Add(h.writeTimeout)); err != nil {
		return err
	}

	return client.conn.WriteJSON(message)
}

func (h *WebSocketHandler) allowConnection(ip string) bool {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	identifier := wsIdentifier(ip)
	if h.clientsByIP[identifier] >= h.server.Config.Server.WSMaxConnectionsIP {
		return false
	}

	h.clientsByIP[identifier]++
	return true
}

func (h *WebSocketHandler) releaseConnection(ip string) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	identifier := wsIdentifier(ip)
	if h.clientsByIP[identifier] > 0 {
		h.clientsByIP[identifier]--
		if h.clientsByIP[identifier] == 0 {
			delete(h.clientsByIP, identifier)
		}
	}
}

func (h *WebSocketHandler) configureConnection(conn *websocket.Conn) {
	conn.SetReadLimit(h.maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(h.pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(h.pongWait))
	})
}

func (h *WebSocketHandler) writePump(client *wsClient, done <-chan struct{}) {
	ticker := time.NewTicker(h.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := h.writeControl(client, websocket.PingMessage, nil); err != nil {
				h.server.Logger.Warn().Err(err).Msg("websocket ping failed")
				return
			}
		}
	}
}

func (h *WebSocketHandler) writeControl(client *wsClient, messageType int, payload []byte) error {
	client.writeMu.Lock()
	defer client.writeMu.Unlock()

	deadline := time.Now().Add(h.writeTimeout)
	return client.conn.WriteControl(messageType, payload, deadline)
}

func (h *WebSocketHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	originHost := originURL.Hostname()
	requestHost := stripPort(r.Host)

	for _, allowedOrigin := range h.server.Config.Server.CORSAllowedOrigins {
		parsedAllowedOrigin, parseErr := url.Parse(allowedOrigin)
		if parseErr == nil && sameOriginHost(parsedAllowedOrigin.Hostname(), originHost) {
			return true
		}

		if allowedOrigin == "*" {
			return true
		}
	}

	return sameOriginHost(originHost, requestHost)
}

func sameOriginHost(originHost, requestHost string) bool {
	return originHost != "" && requestHost != "" && originHost == requestHost
}

func wsIdentifier(ip string) string {
	if ip == "" {
		return "unknown"
	}

	return ip
}

func stripPort(host string) string {
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return parsedHost
	}

	return host
}

func (h *WebSocketHandler) handleMessage(client *wsClient, payload []byte) error {
	var message websocketClientMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return errs.NewBadRequestError("Invalid websocket message payload", false, nil, nil, nil)
	}

	switch message.Type {
	case "subscribe":
		if message.MatchID == nil || *message.MatchID <= 0 {
			return errs.NewBadRequestError("matchId is required", false, nil, nil, nil)
		}
		h.subscribe(client, *message.MatchID)
		matchID := *message.MatchID
		return h.writeJSON(client, websocketMessage{
			Type:    "subscribed",
			MatchID: &matchID,
			Message: "subscribed",
		})
	case "unsubscribe":
		if message.MatchID == nil || *message.MatchID <= 0 {
			return errs.NewBadRequestError("matchId is required", false, nil, nil, nil)
		}
		h.unsubscribe(client, *message.MatchID)
		matchID := *message.MatchID
		return h.writeJSON(client, websocketMessage{
			Type:    "unsubscribed",
			MatchID: &matchID,
			Message: "unsubscribed",
		})
	default:
		return errs.NewBadRequestError("Unsupported websocket message type", false, nil, nil, nil)
	}
}

func (h *WebSocketHandler) subscribe(client *wsClient, matchID int) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	if h.commentarySubscribers[matchID] == nil {
		h.commentarySubscribers[matchID] = make(map[*websocket.Conn]*wsClient)
	}

	h.commentarySubscribers[matchID][client.conn] = client
	client.subscriptions[matchID] = struct{}{}
}

func (h *WebSocketHandler) unsubscribe(client *wsClient, matchID int) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	delete(client.subscriptions, matchID)
	subscribers := h.commentarySubscribers[matchID]
	delete(subscribers, client.conn)
	if len(subscribers) == 0 {
		delete(h.commentarySubscribers, matchID)
	}
}
