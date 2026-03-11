package router

import (
	"github.com/apk471/go-boilerplate/internal/handler"
	"github.com/labstack/echo/v4"
)

func registerMatchRoutes(r *echo.Echo, h *handler.Handlers) {
	matchRouter := r.Group("/matches")
	matchRouter.GET("/", h.Match.GetMatches)
}
