package handler

import (
	"net/http"

	"github.com/apk471/go-boilerplate/internal/model"
	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/apk471/go-boilerplate/internal/service"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type CommentaryHandler struct {
	Handler
	commentaryService *service.CommentaryService
	websocket         *WebSocketHandler
}

type ListCommentaryRequest struct {
	MatchID int `param:"id" validate:"required,gt=0"`
	Limit   int `query:"limit" validate:"omitempty,min=1,max=100"`
}

type CreateCommentaryRequest struct {
	MatchID   int            `param:"id" validate:"required,gt=0"`
	Minute    *int           `json:"minute" validate:"required,gte=0"`
	Sequence  *int           `json:"sequence" validate:"omitempty,gte=0"`
	Period    *string        `json:"period"`
	EventType *string        `json:"eventType"`
	Actor     *string        `json:"actor"`
	Team      *string        `json:"team"`
	Message   string         `json:"message" validate:"required,min=1"`
	Metadata  map[string]any `json:"metadata"`
	Tags      []string       `json:"tags"`
}

func NewCommentaryHandler(s *server.Server, services *service.Services, websocket *WebSocketHandler) *CommentaryHandler {
	return &CommentaryHandler{
		Handler:           NewHandler(s),
		commentaryService: services.Commentary,
		websocket:         websocket,
	}
}

func (r ListCommentaryRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

func (r CreateCommentaryRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

func (h *CommentaryHandler) ListCommentary(c echo.Context) error {
	return Handle(
		h.Handler,
		h.listCommentary,
		http.StatusOK,
		&ListCommentaryRequest{},
	)(c)
}

func (h *CommentaryHandler) CreateCommentary(c echo.Context) error {
	return Handle(
		h.Handler,
		h.createCommentary,
		http.StatusCreated,
		&CreateCommentaryRequest{},
	)(c)
}

func (h *CommentaryHandler) listCommentary(c echo.Context, req *ListCommentaryRequest) ([]model.Commentary, error) {
	return h.commentaryService.ListCommentary(c.Request().Context(), req.MatchID, req.Limit)
}

func (h *CommentaryHandler) createCommentary(c echo.Context, req *CreateCommentaryRequest) (model.Commentary, error) {
	commentary, err := h.commentaryService.CreateCommentary(c.Request().Context(), service.CreateCommentaryInput{
		MatchID:   req.MatchID,
		Minute:    *req.Minute,
		Sequence:  req.Sequence,
		Period:    req.Period,
		EventType: req.EventType,
		Actor:     req.Actor,
		Team:      req.Team,
		Message:   req.Message,
		Metadata:  req.Metadata,
		Tags:      req.Tags,
	})
	if err != nil {
		return model.Commentary{}, err
	}

	if h.websocket != nil {
		h.websocket.BroadcastCommentaryCreated(commentary)
	}

	return commentary, nil
}
