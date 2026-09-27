package domain

import (
	"encoding/json"
	"time"
)

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Analysis struct {
	ID          string          `json:"id"`
	UserID      *string         `json:"userId,omitempty"`
	Source      string          `json:"source"`
	ExternalID  *string         `json:"externalId,omitempty"`
	WhiteName   string          `json:"whiteName"`
	BlackName   string          `json:"blackName"`
	Result      string          `json:"result"`
	PGN         string          `json:"pgn"`
	Status      string          `json:"status"`
	Progress    int             `json:"progress"`
	Error       *string         `json:"error,omitempty"`
	Summary     json.RawMessage `json:"summary,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	CompletedAt *time.Time      `json:"completedAt,omitempty"`
	Moves       []MoveAnalysis  `json:"moves,omitempty"`
}

type MoveAnalysis struct {
	Ply                int      `json:"ply"`
	SAN                string   `json:"san"`
	UCI                string   `json:"uci"`
	FENBefore          string   `json:"fenBefore"`
	FENAfter           string   `json:"fenAfter"`
	ScoreBefore        *int     `json:"scoreBefore,omitempty"`
	ScoreAfter         *int     `json:"scoreAfter,omitempty"`
	MateBefore         *int     `json:"mateBefore,omitempty"`
	MateAfter          *int     `json:"mateAfter,omitempty"`
	BestMove           string   `json:"bestMove"`
	PrincipalVariation []string `json:"principalVariation"`
	Classification     string   `json:"classification"`
	Loss               int      `json:"loss"`
	Explanation        string   `json:"explanation"`
}

type GamePreview struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	White     string    `json:"white"`
	Black     string    `json:"black"`
	WhiteElo  int       `json:"whiteElo,omitempty"`
	BlackElo  int       `json:"blackElo,omitempty"`
	Result    string    `json:"result"`
	PlayedAt  time.Time `json:"playedAt"`
	TimeClass string    `json:"timeClass,omitempty"`
	Opening   string    `json:"opening,omitempty"`
	PGN       string    `json:"pgn"`
}
