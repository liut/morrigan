package convo

import (
	"math"
	"testing"
	"time"
)

func TestCalcRetention(t *testing.T) {
	if r := CalcRetention(time.Time{}, 1.0); r != 1.0 {
		t.Errorf("zero lastAccessedAt → want R=1.0, got %.3f", r)
	}

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	r := CalcRetention(oneHourAgo, 1.0)
	if r < 0.95 || r > 0.97 {
		t.Errorf("1h ago, decay=1 → want R≈0.959, got %.3f", r)
	}

	oneDayAgo := time.Now().Add(-24 * time.Hour)
	r = CalcRetention(oneDayAgo, 7.0)
	if r < 0.85 || r > 0.88 {
		t.Errorf("24h ago, decay=7 → want R≈0.867, got %.3f", r)
	}

	r = CalcRetention(oneDayAgo, 60.0)
	if r < 0.97 || r > 0.99 {
		t.Errorf("24h ago, decay=60 → want R≈0.983, got %.3f", r)
	}

	r = CalcRetention(oneHourAgo, 0)
	if r < 0.95 || r > 0.97 {
		t.Errorf("decay=0 → fallback to 1.0, got %.3f", r)
	}
}

func TestCalcReinforce(t *testing.T) {
	r := CalcReinforce(0.5, 0.3)
	if math.Abs(r-0.65) > 0.01 {
		t.Errorf("R=0.5, F=0.3 → want 0.65, got %.3f", r)
	}
	if r := CalcReinforce(1.0, 0.3); r != 1.0 {
		t.Errorf("R=1.0 → want 1.0, got %.3f", r)
	}
	r = CalcReinforce(0.9, 0.5)
	if math.Abs(r-0.95) > 0.01 {
		t.Errorf("R=0.9, F=0.5 → want 0.95, got %.3f", r)
	}
}

func TestTierDecayRate(t *testing.T) {
	tests := []struct {
		tier string
		want float64
	}{
		{"working", 1.0},
		{"short-term", 7.0},
		{"long-term", 60.0},
		{"unknown", 1.0},
		{"", 1.0},
	}
	for _, tt := range tests {
		if got := TierDecayRate(tt.tier); got != tt.want {
			t.Errorf("TierDecayRate(%q) = %.1f, want %.1f", tt.tier, got, tt.want)
		}
	}
}

func TestForgottenMultiplier(t *testing.T) {
	if fm := ForgottenMultiplier(0.1, 0.05); fm != 1.0 {
		t.Errorf("R=0.1 > threshold → want 1.0, got %.2f", fm)
	}
	if fm := ForgottenMultiplier(0.04, 0.05); fm != 0.1 {
		t.Errorf("R=0.04 < threshold → want 0.1, got %.2f", fm)
	}
	if fm := ForgottenMultiplier(0.05, 0.05); fm != 1.0 {
		t.Errorf("R=0.05 = threshold → want 1.0, got %.2f", fm)
	}
}

func TestRouteTier(t *testing.T) {
	if got := RouteTier(0.9, 0.8, 0.6); got != "long-term" {
		t.Errorf("score 0.9, thresholds 0.8/0.6 → want long-term, got %s", got)
	}
	if got := RouteTier(0.7, 0.8, 0.6); got != "short-term" {
		t.Errorf("score 0.7 → want short-term, got %s", got)
	}
	if got := RouteTier(0.3, 0.8, 0.6); got != "working" {
		t.Errorf("score 0.3 → want working, got %s", got)
	}
}

func TestPromoteTier(t *testing.T) {
	if got := PromoteTier("working", 3, 3, 10); got != "short-term" {
		t.Errorf("working, access=3 → want short-term, got %s", got)
	}
	if got := PromoteTier("working", 2, 3, 10); got != "" {
		t.Errorf("working, access=2 → want '', got %s", got)
	}
	if got := PromoteTier("short-term", 10, 3, 10); got != "long-term" {
		t.Errorf("short-term, access=10 → want long-term, got %s", got)
	}
	if got := PromoteTier("long-term", 20, 3, 10); got != "" {
		t.Errorf("long-term → want '', got %s", got)
	}
}
