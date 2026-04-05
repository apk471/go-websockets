package handler

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/apk471/go-boilerplate/internal/errs"
	appMiddleware "github.com/apk471/go-boilerplate/internal/middleware"
	"github.com/apk471/go-boilerplate/internal/model"
	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type wsClient struct {
	conn    *websocket.Conn
	ip      string
	writeMu sync.Mutex
}

type websocketMessage struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type WebSocketHandler struct {
	server         *server.Server
	rateLimit      *appMiddleware.RateLimitMiddleware
	upgrader       websocket.Upgrader
	clients        map[*websocket.Conn]*wsClient
	clientsByIP    map[string]int
	clientsMu      sync.RWMutex
	pingInterval   time.Duration
	pongWait       time.Duration
	writeTimeout   time.Duration
	maxMessageSize int64
}

func NewWebSocketHandler(s *server.Server) *WebSocketHandler {
	handler := &WebSocketHandler{
		server:         s,
		rateLimit:      appMiddleware.NewRateLimitMiddleware(s),
		clients:        make(map[*websocket.Conn]*wsClient),
		clientsByIP:    make(map[string]int),
		pingInterval:   time.Duration(s.Config.Server.WSPingInterval) * time.Second,
		pongWait:       time.Duration(s.Config.Server.WSPongWait) * time.Second,
		writeTimeout:   time.Duration(s.Config.Server.WSWriteTimeout) * time.Second,
		maxMessageSize: s.Config.Server.WSMaxMessageBytes,
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

	client := &wsClient{conn: conn, ip: wsIdentifier(ip)}
	h.addClient(client)
	defer h.removeClient(conn, client.ip)

	h.configureConnection(conn)

	done := make(chan struct{})
	go h.writePump(client, done)
	defer close(done)

	h.server.Logger.Info().Msg("websocket client connected")

	if err := h.writeJSON(client, websocketMessage{
		Type: "welcome",
		Data: map[string]string{
			"message": "welcome",
		},
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

		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}

		if !h.rateLimit.AllowWSMessage(ip) {
			h.rateLimit.RecordRateLimitHit("websocket_message", c.Path(), ip)
			_ = h.writeControl(client, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "message rate limit exceeded"))
			break
		}

		h.server.Logger.Warn().
			Int("bytes", len(payload)).
			Str("ip", ip).
			Msg("dropping unsupported websocket client message")

		_ = h.writeControl(client, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "client messages are not supported"))
		break
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
	h.Broadcast(websocketMessage{
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
	delete(h.clients, conn)
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
