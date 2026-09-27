package chess

import (
	"fmt"
	"io"
	"strings"

	"github.com/ahror/royal-move/internal/domain"
	notchess "github.com/notnil/chess"
)

type ParsedGame struct {
	White  string
	Black  string
	Result string
	PGN    string
	Game   *notchess.Game
}

func ParsePGN(raw string) (ParsedGame, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return ParsedGame{}, fmt.Errorf("PGN text is empty")
	}
	if !hasResult(clean) {
		clean += " *"
	}
	option, err := notchess.PGN(strings.NewReader(clean))
	if err != nil {
		return ParsedGame{}, fmt.Errorf("PGN could not be parsed: %w", err)
	}
	game := notchess.NewGame(option)
	if len(game.Moves()) == 0 {
		return ParsedGame{}, fmt.Errorf("no moves were found in the PGN")
	}
	white, black, result := "White", "Black", "*"
	for _, tag := range game.TagPairs() {
		switch tag.Key {
		case "White":
			white = tag.Value
		case "Black":
			black = tag.Value
		case "Result":
			result = tag.Value
		}
	}
	return ParsedGame{White: white, Black: black, Result: result, PGN: clean, Game: game}, nil
}

func ParseMany(reader io.Reader) ([]ParsedGame, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	blocks := splitGames(string(data))
	games := make([]ParsedGame, 0, len(blocks))
	for _, block := range blocks {
		game, parseErr := ParsePGN(block)
		if parseErr == nil {
			games = append(games, game)
		}
	}
	if len(games) == 0 {
		return nil, fmt.Errorf("no valid games were found")
	}
	return games, nil
}

func Positions(game *notchess.Game) ([]domain.MoveAnalysis, error) {
	replay := notchess.NewGame()
	notation := notchess.AlgebraicNotation{}
	moves := make([]domain.MoveAnalysis, 0, len(game.Moves()))
	for index, move := range game.Moves() {
		before := replay.Position().String()
		san := notation.Encode(replay.Position(), move)
		uci := move.String()
		if err := replay.Move(move); err != nil {
			return nil, fmt.Errorf("move %d is illegal: %w", index+1, err)
		}
		moves = append(moves, domain.MoveAnalysis{Ply: index + 1, SAN: san, UCI: uci, FENBefore: before, FENAfter: replay.Position().String()})
	}
	return moves, nil
}

func hasResult(pgn string) bool {
	fields := strings.Fields(pgn)
	if len(fields) == 0 {
		return false
	}
	last := fields[len(fields)-1]
	return last == "1-0" || last == "0-1" || last == "1/2-1/2" || last == "*"
}

func splitGames(input string) []string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	parts := strings.Split(input, "\n\n[Event ")
	for i := 1; i < len(parts); i++ {
		parts[i] = "[Event " + parts[i]
	}
	return parts
}
