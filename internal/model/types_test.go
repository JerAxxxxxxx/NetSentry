package model

import "testing"

func TestLossRate(t *testing.T) {
	stats := MonitorStats{
		Total:       10,
		Timeout:     2,
		Unreachable: 1,
		Errors:      1,
	}

	got := stats.LossRate()
	want := 40.0
	if got != want {
		t.Fatalf("expected loss rate %.2f, got %.2f", want, got)
	}
}

func TestLossRateZeroWhenNoProbes(t *testing.T) {
	if got := (MonitorStats{}).LossRate(); got != 0 {
		t.Fatalf("expected zero loss rate, got %.2f", got)
	}
}
