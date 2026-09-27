package chess

import "testing"

func TestParsePartialMoveText(t *testing.T) {
	raw := "1.e4\ne5\n2.Nf3\nNc6\n3.Bb5\na6\n4.Ba4\nNf6\n5.d3"
	parsed, err := ParsePGN(raw)
	if err != nil {
		t.Fatalf("ParsePGN() error = %v", err)
	}
	if got := len(parsed.Game.Moves()); got != 9 {
		t.Fatalf("moves = %d, want 9", got)
	}
	positions, err := Positions(parsed.Game)
	if err != nil {
		t.Fatalf("Positions() error = %v", err)
	}
	if positions[0].SAN != "e4" || positions[8].SAN != "d3" {
		t.Fatalf("unexpected SAN sequence: %s ... %s", positions[0].SAN, positions[8].SAN)
	}
}

func TestRejectsIllegalMove(t *testing.T) {
	if _, err := ParsePGN("1. e4 e5 2. Bh8"); err == nil {
		t.Fatal("illegal move was accepted")
	}
}
