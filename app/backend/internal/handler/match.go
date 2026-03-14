package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/apk471/go-boilerplate/internal/model"
	"github.com/apk471/go-boilerplate/internal/server"
	"github.com/apk471/go-boilerplate/internal/service"
	"github.com/apk471/go-boilerplate/internal/validation"
	"github.com/labstack/echo/v4"
	"github.com/go-playground/validator/v10"
)

type MatchHandler struct {
	Handler
	matchService *service.MatchService
}

type CreateMatchRequest struct {
	Sport     string     `json:"sport" validate:"required,min=2,max=100"`
	HomeTeam  string     `json:"homeTeam" validate:"required,min=2,max=100"`
	AwayTeam  string     `json:"awayTeam" validate:"required,min=2,max=100"`
	StartTime *time.Time `json:"startTime" validate:"required"`
	EndTime   *time.Time `json:"endTime" validate:"required"`
}

type ListMatchesRequest struct {
	Limit int `query:"limit" validate:"omitempty,min=1,max=100"`
}

func (r CreateMatchRequest) Validate() error {
	validate := validator.New()
	if err := validate.Struct(r); err != nil {
		return err
	}

	var errs validation.CustomValidationErrors
	if strings.EqualFold(strings.TrimSpace(r.HomeTeam), strings.TrimSpace(r.AwayTeam)) {
		errs = append(errs, validation.CustomValidationError{
			Field:   "awayTeam",
			Message: "must be different from homeTeam",
		})
	}

	if r.StartTime != nil && r.EndTime != nil && !r.EndTime.After(*r.StartTime) {
		errs = append(errs, validation.CustomValidationError{
			Field:   "endTime",
			Message: "must be after startTime",
		})
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func (r ListMatchesRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

func NewMatchHandler(s *server.Server, services *service.Services) *MatchHandler {
	return &MatchHandler{
		Handler:      NewHandler(s),
		matchService: services.Match,
	}
}

func (h *MatchHandler) GetMatches(c echo.Context) error {
	return Handle(
		h.Handler,
		h.listMatches,
		http.StatusOK,
		&ListMatchesRequest{},
	)(c)
}

func (h *MatchHandler) CreateMatch(c echo.Context) error {
	return Handle(
		h.Handler,
		h.createMatch,
		http.StatusCreated,
		&CreateMatchRequest{},
	)(c)
}

func (h *MatchHandler) createMatch(c echo.Context, req *CreateMatchRequest) (model.Match, error) {
	return h.matchService.CreateMatch(c.Request().Context(), service.CreateMatchInput{
		Sport:     req.Sport,
		HomeTeam:  req.HomeTeam,
		AwayTeam:  req.AwayTeam,
		StartTime: *req.StartTime,
		EndTime:   *req.EndTime,
	})
}

func (h *MatchHandler) listMatches(c echo.Context, req *ListMatchesRequest) ([]model.Match, error) {
	return h.matchService.ListMatches(c.Request().Context(), req.Limit)
}
