package engine

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ahror/royal-move/internal/domain"
)

type Evaluation struct {
	Score    *int
	Mate     *int
	BestMove string
	PV       []string
	Depth    int
}

type Stockfish struct {
	cmd      *exec.Cmd
	in       io.WriteCloser
	lines    chan string
	done     chan error
	moveTime time.Duration
}

func Start(path string, threads, hashMB int, moveTime time.Duration) (*Stockfish, error) {
	cmd := exec.Command(path)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stderr, _ := cmd.StderrPipe()
	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("Stockfish ishga tushmadi: %w", err)
	}
	s := &Stockfish{cmd: cmd, in: in, lines: make(chan string, 256), done: make(chan error, 1), moveTime: moveTime}
	go s.scan(out)
	go io.Copy(io.Discard, stderr)
	s.send("uci")
	if err = s.waitFor(context.Background(), "uciok", 10*time.Second); err != nil {
		s.Close()
		return nil, err
	}
	s.send(fmt.Sprintf("setoption name Threads value %d", threads))
	s.send(fmt.Sprintf("setoption name Hash value %d", hashMB))
	s.send("setoption name MultiPV value 1")
	s.send("isready")
	if err = s.waitFor(context.Background(), "readyok", 10*time.Second); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func (s *Stockfish) scan(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		s.lines <- scanner.Text()
	}
	s.done <- scanner.Err()
	close(s.lines)
}
func (s *Stockfish) send(command string) { _, _ = io.WriteString(s.in, command+"\n") }
func (s *Stockfish) waitFor(ctx context.Context, target string, limit time.Duration) error {
	timer := time.NewTimer(limit)
	defer timer.Stop()
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				return fmt.Errorf("Stockfish yopildi")
			}
			if line == target {
				return nil
			}
		case err := <-s.done:
			return err
		case <-timer.C:
			return fmt.Errorf("Stockfish javobi kutilgan vaqtda kelmadi")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
func (s *Stockfish) Close() {
	s.send("quit")
	_ = s.in.Close()
	done := make(chan struct{})
	go func() { _ = s.cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = s.cmd.Process.Kill()
	}
}

func (s *Stockfish) Evaluate(ctx context.Context, fen string) (Evaluation, error) {
	s.send("position fen " + fen)
	s.send(fmt.Sprintf("go movetime %d", s.moveTime.Milliseconds()))
	evaluation := Evaluation{}
	timer := time.NewTimer(s.moveTime + 10*time.Second)
	defer timer.Stop()
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				return evaluation, fmt.Errorf("Stockfish yopildi")
			}
			if strings.HasPrefix(line, "info ") {
				parseInfo(line, &evaluation)
			}
			if strings.HasPrefix(line, "bestmove ") {
				parts := strings.Fields(line)
				if len(parts) > 1 {
					evaluation.BestMove = parts[1]
				}
				return evaluation, nil
			}
		case <-timer.C:
			s.send("stop")
			return evaluation, fmt.Errorf("position analysis timed out")
		case <-ctx.Done():
			s.send("stop")
			return evaluation, ctx.Err()
		}
	}
}

func parseInfo(line string, target *Evaluation) {
	parts := strings.Fields(line)
	depth := fieldInt(parts, "depth")
	if depth < target.Depth {
		return
	}
	scoreIndex := indexOf(parts, "score")
	if scoreIndex < 0 || scoreIndex+2 >= len(parts) {
		return
	}
	value, err := strconv.Atoi(parts[scoreIndex+2])
	if err != nil {
		return
	}
	target.Depth = depth
	if parts[scoreIndex+1] == "cp" {
		target.Score = &value
		target.Mate = nil
	} else if parts[scoreIndex+1] == "mate" {
		target.Mate = &value
		target.Score = nil
	}
	if pv := indexOf(parts, "pv"); pv >= 0 && pv+1 < len(parts) {
		target.PV = append([]string(nil), parts[pv+1:]...)
	}
}
func fieldInt(parts []string, name string) int {
	i := indexOf(parts, name)
	if i < 0 || i+1 >= len(parts) {
		return 0
	}
	v, _ := strconv.Atoi(parts[i+1])
	return v
}
func indexOf(parts []string, value string) int {
	for i, item := range parts {
		if item == value {
			return i
		}
	}
	return -1
}

func Analyze(ctx context.Context, s *Stockfish, moves []domain.MoveAnalysis, progress func(int)) ([]domain.MoveAnalysis, map[string]any, error) {
	if len(moves) == 0 {
		return nil, nil, fmt.Errorf("no moves were found")
	}
	before, err := s.Evaluate(ctx, moves[0].FENBefore)
	if err != nil {
		return nil, nil, err
	}
	totalLoss := 0
	counts := map[string]int{}
	for i := range moves {
		after, evalErr := s.Evaluate(ctx, moves[i].FENAfter)
		if evalErr != nil {
			return nil, nil, evalErr
		}
		loss := lossFor(before, after)
		class := classify(loss, before, after, moves[i].UCI)
		moves[i].ScoreBefore = before.Score
		moves[i].MateBefore = before.Mate
		moves[i].ScoreAfter = negate(after.Score)
		moves[i].MateAfter = negate(after.Mate)
		moves[i].BestMove = before.BestMove
		moves[i].PrincipalVariation = before.PV
		moves[i].Loss = loss
		moves[i].Classification = class
		moves[i].Explanation = explain(class, loss)
		totalLoss += loss
		counts[class]++
		before = after
		progress((i + 1) * 100 / len(moves))
	}
	accuracy := 100 * math.Exp(-float64(totalLoss)/float64(max(1, len(moves))*220))
	summary := map[string]any{"accuracy": math.Round(accuracy*10) / 10, "averageLoss": totalLoss / max(1, len(moves)), "classifications": counts, "totalMoves": len(moves)}
	return moves, summary, nil
}

func lossFor(best, after Evaluation) int {
	if best.Mate != nil {
		if *best.Mate > 0 && (after.Mate == nil || *after.Mate >= 0) {
			return 1200
		}
		return 0
	}
	if best.Score == nil {
		return 0
	}
	actual := 0
	if after.Score != nil {
		actual = -*after.Score
	} else if after.Mate != nil {
		if *after.Mate > 0 {
			actual = -10000
		} else {
			actual = 10000
		}
	}
	return max(0, *best.Score-actual)
}
func classify(loss int, best, after Evaluation, played string) string {
	if played == best.BestMove {
		return "precise"
	}
	switch {
	case loss <= 10:
		return "strong"
	case loss <= 35:
		return "good"
	case loss <= 80:
		return "acceptable"
	case loss <= 150:
		return "dubious"
	case loss <= 300:
		return "mistake"
	default:
		return "critical"
	}
}
func explain(class string, loss int) string {
	switch class {
	case "precise":
		return "The strongest continuation in the position was played."
	case "strong":
		return "A strong move, very close to the engine's first choice."
	case "good":
		return "A good move that preserved the balance of the position."
	case "acceptable":
		return "A playable move, although a more precise continuation was available."
	case "dubious":
		return fmt.Sprintf("Part of the advantage was lost (%0.2f pawns).", float64(loss)/100)
	case "mistake":
		return "A mistake that significantly worsened the position."
	default:
		return "A critical error that changed the course of the game."
	}
}
func negate(value *int) *int {
	if value == nil {
		return nil
	}
	v := -*value
	return &v
}
