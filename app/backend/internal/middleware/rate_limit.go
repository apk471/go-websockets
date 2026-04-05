package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type RateLimitMiddleware struct {
	server           *server.Server
	httpLimiter      *ipRateLimiter
	wsUpgradeLimiter *ipRateLimiter
	wsMessageLimiter *ipRateLimiter
}

func NewRateLimitMiddleware(s *server.Server) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		server:           s,
		httpLimiter:      newIPRateLimiter(s.Config.Server.HTTPRateLimitRPS, s.Config.Server.HTTPRateLimitBurst, time.Duration(s.Config.Server.HTTPRateLimitTTL)*time.Second),
		wsUpgradeLimiter: newIPRateLimiter(s.Config.Server.WSUpgradeRateRPS, s.Config.Server.WSUpgradeRateBurst, time.Duration(s.Config.Server.HTTPRateLimitTTL)*time.Second),
		wsMessageLimiter: newIPRateLimiter(s.Config.Server.WSMessageRateRPS, s.Config.Server.WSMessageRateBurst, time.Duration(s.Config.Server.HTTPRateLimitTTL)*time.Second),
	}
}

func (r *RateLimitMiddleware) HTTP() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Path() == "/ws" {
				return next(c)
			}

			if r.httpLimiter.Allow(clientIdentifier(c.RealIP())) {
				return next(c)
			}

			r.RecordRateLimitHit("http", c.Path(), c.RealIP())

			r.server.Logger.Warn().
				Str("request_id", GetRequestID(c)).
				Str("path", c.Path()).
				Str("method", c.Request().Method).
				Str("ip", c.RealIP()).
				Msg("http rate limit exceeded")

			return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
		}
	}
}

func (r *RateLimitMiddleware) AllowWSUpgrade(identifier string) bool {
	return r.wsUpgradeLimiter.Allow(clientIdentifier(identifier))
}

func (r *RateLimitMiddleware) AllowWSMessage(identifier string) bool {
	return r.wsMessageLimiter.Allow(clientIdentifier(identifier))
}

func (r *RateLimitMiddleware) RecordRateLimitHit(kind, endpoint, identifier string) {
	if r.server.LoggerService != nil && r.server.LoggerService.GetApplication() != nil {
		r.server.LoggerService.GetApplication().RecordCustomEvent("RateLimitHit", map[string]interface{}{
			"kind":       kind,
			"endpoint":   endpoint,
			"identifier": clientIdentifier(identifier),
		})
	}
}

type ipRateLimiter struct {
	rate      rate.Limit
	burst     int
	ttl       time.Duration
	mu        sync.Mutex
	visitors  map[string]*visitor
	lastSweep time.Time
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter(requestsPerSecond float64, burst int, ttl time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		rate:      rate.Limit(requestsPerSecond),
		burst:     burst,
		ttl:       ttl,
		visitors:  make(map[string]*visitor),
		lastSweep: time.Now(),
	}
}

func (l *ipRateLimiter) Allow(identifier string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastSweep) >= l.ttl {
		l.cleanup(now)
	}

	entry, ok := l.visitors[identifier]
	if !ok {
		entry = &visitor{
			limiter: rate.NewLimiter(l.rate, l.burst),
		}
		l.visitors[identifier] = entry
	}

	entry.lastSeen = now
	return entry.limiter.Allow()
}

func (l *ipRateLimiter) cleanup(now time.Time) {
	for identifier, entry := range l.visitors {
		if now.Sub(entry.lastSeen) > l.ttl {
			delete(l.visitors, identifier)
		}
	}
	l.lastSweep = now
}

func clientIdentifier(identifier string) string {
	if identifier == "" {
		return "unknown"
	}

	return identifier
}
