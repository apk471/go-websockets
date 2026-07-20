package router

import (
	"github.com/ayush-amin/go-boilerplate/internal/handler"
	"github.com/labstack/echo/v4"
)

func registerMatchRoutes(r *echo.Echo, h *handler.Handlers) {
	matchRouter := r.Group("/matches")
	matchRouter.GET("/", h.Match.GetMatches)
	matchRouter.POST("", h.Match.CreateMatch)

	commentaryRouter := matchRouter.Group("/:id/commentary")
	commentaryRouter.GET("", h.Commentary.ListCommentary)
	commentaryRouter.POST("", h.Commentary.CreateCommentary)
}
