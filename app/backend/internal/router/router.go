package router

import (
	"github.com/apk471/go-boilerplate/internal/handler"
	"github.com/apk471/go-boilerplate/internal/middleware"
	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/apk471/go-boilerplate/internal/service"
	"github.com/labstack/echo/v4"
)

func NewRouter(s *server.Server, h *handler.Handlers, services *service.Services) *echo.Echo {
	middlewares := middleware.NewMiddlewares(s)

	router := echo.New()

	router.HTTPErrorHandler = middlewares.Global.GlobalErrorHandler

	// global middlewares
	router.Use(
		middleware.RequestID(),
		middlewares.RateLimit.HTTP(),
		middlewares.Global.CORS(),
		middlewares.Global.Secure(),
		middlewares.Global.BodyLimit(),
		middlewares.Tracing.NewRelicMiddleware(),
		middlewares.Tracing.EnhanceTracing(),
		middlewares.ContextEnhancer.EnhanceContext(),
		middlewares.Global.RequestLogger(),
		middlewares.Global.Recover(),
	)

	// register system routes
	registerSystemRoutes(router, h)
	registerMatchRoutes(router, h)

	// register versioned routes
	router.Group("/api/v1")

	return router
}
