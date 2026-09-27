package platform

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	chessparser "github.com/ahror/royal-move/internal/chess"
	"github.com/ahror/royal-move/internal/domain"
)

type Client struct{ HTTP *http.Client }

func New() *Client { return &Client{HTTP: &http.Client{Timeout: 25 * time.Second}} }

func (c *Client) Games(ctx context.Context, source, username string) ([]domain.GamePreview, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("enter a username")
	}
	switch source {
	case "chesscom":
		return c.chessCom(ctx, username)
	case "lichess":
		return c.lichess(ctx, username)
	default:
		return nil, fmt.Errorf("unsupported platform")
	}
}

func (c *Client) chessCom(ctx context.Context, username string) ([]domain.GamePreview, error) {
	archiveURL := "https://api.chess.com/pub/player/" + url.PathEscape(strings.ToLower(username)) + "/games/archives"
	var archives struct {
		Archives []string `json:"archives"`
	}
	if err := c.getJSON(ctx, archiveURL, &archives); err != nil {
		return nil, err
	}
	if len(archives.Archives) == 0 {
		return []domain.GamePreview{}, nil
	}
	var payload struct {
		Games []struct {
			URL          string `json:"url"`
			PGN          string `json:"pgn"`
			TimeClass    string `json:"time_class"`
			EndTime      int64  `json:"end_time"`
			White, Black struct {
				Name   string `json:"username"`
				Rating int    `json:"rating"`
				Result string `json:"result"`
			}
		} `json:"games"`
	}
	if err := c.getJSON(ctx, archives.Archives[len(archives.Archives)-1], &payload); err != nil {
		return nil, err
	}
	items := make([]domain.GamePreview, 0, len(payload.Games))
	for _, g := range payload.Games {
		result := "1/2-1/2"
		if g.White.Result == "win" {
			result = "1-0"
		} else if g.Black.Result == "win" {
			result = "0-1"
		}
		id := g.URL[strings.LastIndex(g.URL, "/")+1:]
		items = append(items, domain.GamePreview{ID: id, Source: "chesscom", White: g.White.Name, Black: g.Black.Name, WhiteElo: g.White.Rating, BlackElo: g.Black.Rating, Result: result, PlayedAt: time.Unix(g.EndTime, 0), TimeClass: g.TimeClass, PGN: g.PGN})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].PlayedAt.After(items[j].PlayedAt) })
	if len(items) > 30 {
		items = items[:30]
	}
	return items, nil
}

func (c *Client) lichess(ctx context.Context, username string) ([]domain.GamePreview, error) {
	endpoint := "https://lichess.org/api/games/user/" + url.PathEscape(username) + "?max=30&pgnInJson=true&clocks=false&evals=false&opening=true"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Accept", "application/x-ndjson")
	req.Header.Set("User-Agent", "RoyalMove/1.0")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, apiError("Lichess", resp)
	}
	items := []domain.GamePreview{}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 2*1024*1024)
	for scanner.Scan() {
		var g struct {
			ID, Speed, Moves string
			Created          int64 `json:"createdAt"`
			Players          struct {
				White, Black struct {
					User   struct{ Name string }
					Rating int
				}
			}
			Winner  string
			Opening struct{ Name string }
		}
		if json.Unmarshal(scanner.Bytes(), &g) != nil {
			continue
		}
		result := "1/2-1/2"
		if g.Winner == "white" {
			result = "1-0"
		} else if g.Winner == "black" {
			result = "0-1"
		}
		if g.Moves == "" {
			continue
		}
		pgn := fmt.Sprintf("[Event \"Lichess game\"]\n[Site \"https://lichess.org/%s\"]\n[Date \"%s\"]\n[White \"%s\"]\n[Black \"%s\"]\n[Result \"%s\"]\n\n%s %s",
			g.ID, time.UnixMilli(g.Created).UTC().Format("2006.01.02"), safeTag(g.Players.White.User.Name), safeTag(g.Players.Black.User.Name), result, g.Moves, result)
		items = append(items, domain.GamePreview{ID: g.ID, Source: "lichess", White: g.Players.White.User.Name, Black: g.Players.Black.User.Name, WhiteElo: g.Players.White.Rating, BlackElo: g.Players.Black.Rating, Result: result, PlayedAt: time.UnixMilli(g.Created), TimeClass: g.Speed, Opening: g.Opening.Name, PGN: pgn})
	}
	return items, scanner.Err()
}

func safeTag(value string) string { return strings.ReplaceAll(value, "\"", "'") }

func (c *Client) getJSON(ctx context.Context, endpoint string, target any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("User-Agent", "RoyalMove/1.0 (contact: admin@royalmove.local)")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return apiError("Chess.com", resp)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
func apiError(name string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode == 404 {
		return fmt.Errorf("%s user not found", name)
	}
	if resp.StatusCode == 429 {
		return fmt.Errorf("%s rate limit reached; please try again shortly", name)
	}
	return fmt.Errorf("%s API error (%d): %s", name, resp.StatusCode, strings.TrimSpace(string(body)))
}

var _ = chessparser.ParsePGN
