package engine

import "testing"

func TestClassification(t *testing.T) {
	tests := []struct {
		loss int
		want string
	}{{0, "strong"}, {20, "good"}, {60, "acceptable"}, {120, "dubious"}, {240, "mistake"}, {450, "critical"}}
	for _, tt := range tests {
		if got := classify(tt.loss, Evaluation{BestMove: "e2e4"}, Evaluation{}, "d2d4"); got != tt.want {
			t.Errorf("classify(%d)=%s, want %s", tt.loss, got, tt.want)
		}
	}
	if got := classify(500, Evaluation{BestMove: "e2e4"}, Evaluation{}, "e2e4"); got != "precise" {
		t.Errorf("best move = %s", got)
	}
}
