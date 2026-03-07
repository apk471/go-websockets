package model

import "time"

type MatchStatus string

const (
	MatchStatusScheduled MatchStatus = "scheduled"
	MatchStatusLive      MatchStatus = "live"
	MatchStatusFinished  MatchStatus = "finished"
)

type Match struct {
	ID        int         `json:"id" db:"id"`
	Sport     string      `json:"sport" db:"sport"`
	HomeTeam  string      `json:"homeTeam" db:"home_team"`
	AwayTeam  string      `json:"awayTeam" db:"away_team"`
	Status    MatchStatus `json:"status" db:"status"`
	StartTime *time.Time  `json:"startTime,omitempty" db:"start_time"`
	EndTime   *time.Time  `json:"endTime,omitempty" db:"end_time"`
	HomeScore int         `json:"homeScore" db:"home_score"`
	AwayScore int         `json:"awayScore" db:"away_score"`
	CreatedAt time.Time   `json:"createdAt" db:"created_at"`
}

type Commentary struct {
	ID        int            `json:"id" db:"id"`
	MatchID   int            `json:"matchId" db:"match_id"`
	Minute    *int           `json:"minute,omitempty" db:"minute"`
	Sequence  *int           `json:"sequence,omitempty" db:"sequence"`
	Period    *string        `json:"period,omitempty" db:"period"`
	EventType *string        `json:"eventType,omitempty" db:"event_type"`
	Actor     *string        `json:"actor,omitempty" db:"actor"`
	Team      *string        `json:"team,omitempty" db:"team"`
	Message   string         `json:"message" db:"message"`
	Metadata  map[string]any `json:"metadata,omitempty" db:"metadata"`
	Tags      []string       `json:"tags,omitempty" db:"tags"`
	CreatedAt time.Time      `json:"createdAt" db:"created_at"`
}
