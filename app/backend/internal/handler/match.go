package handler

import (
	"net/http"

	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/labstack/echo/v4"
)

type MatchHandler struct {
	Handler
}

func NewMatchHandler(s *server.Server) *MatchHandler {
	return &MatchHandler{
		Handler: NewHandler(s),
	}
}

func (h *MatchHandler) GetMatches(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"message": "working",
	})
}
