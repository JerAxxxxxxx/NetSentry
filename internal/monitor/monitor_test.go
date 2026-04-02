package monitor

import (
	"testing"
	"time"

	"netsentry/internal/model"
)

func TestUpdateStatsTracksSuccessfulProbeMetrics(t *testing.T) {
	stats := model.MonitorStats{
		Host:        "www.baidu.com",
		ResolvedIPs: map[string]int{},
	}
	var firstIssueAt time.Time
	now := time.Date(2026, 3, 26, 10, 0, 0, 0, time.Local)

	updateStats(&stats, model.ProbeResult{
		Time:         now,
		Host:         "www.baidu.com",
		IP:           "183.2.172.177",
		Status:       model.StatusOK,
		RTT:          8 * time.Millisecond,
		TotalLatency: 10 * time.Millisecond,
	}, &firstIssueAt)

	if stats.Total != 1 || stats.Success != 1 {
		t.Fatalf("expected one successful probe, got total=%d success=%d", stats.Total, stats.Success)
	}
	if stats.RTTSum != 8*time.Millisecond || stats.MaxRTT != 8*time.Millisecond {
		t.Fatalf("unexpected RTT aggregation: sum=%s max=%s", stats.RTTSum, stats.MaxRTT)
	}
	if stats.ResolvedIPs["183.2.172.177"] != 1 {
		t.Fatalf("expected resolved IP count to be tracked")
	}
	if !stats.FirstProbe.Equal(now) || !stats.LastProbe.Equal(now) {
		t.Fatalf("expected probe timestamps to be initialized")
	}
}

func TestUpdateStatsTracksTimeoutStreakAndFirstIssue(t *testing.T) {
	stats := model.MonitorStats{
		Host:        "www.baidu.com",
		ResolvedIPs: map[string]int{},
	}
	var firstIssueAt time.Time
	base := time.Date(2026, 3, 26, 10, 0, 0, 0, time.Local)

	updateStats(&stats, model.ProbeResult{
		Time:         base,
		Host:         "www.baidu.com",
		IP:           "183.2.172.177",
		Status:       model.StatusTimeout,
		TotalLatency: 2 * time.Second,
	}, &firstIssueAt)
	updateStats(&stats, model.ProbeResult{
		Time:         base.Add(time.Second),
		Host:         "www.baidu.com",
		IP:           "183.2.172.177",
		Status:       model.StatusTimeout,
		TotalLatency: 2 * time.Second,
	}, &firstIssueAt)

	if stats.Timeout != 2 {
		t.Fatalf("expected 2 timeouts, got %d", stats.Timeout)
	}
	if stats.ConsecutiveIssues != 2 || stats.CurrentTimeouts != 2 || stats.MaxTimeoutStreak != 2 {
		t.Fatalf("unexpected timeout streak tracking: consecutive=%d current=%d max=%d", stats.ConsecutiveIssues, stats.CurrentTimeouts, stats.MaxTimeoutStreak)
	}
	if firstIssueAt.IsZero() || !firstIssueAt.Equal(base) {
		t.Fatalf("expected first issue timestamp to be set to first timeout")
	}
}

func TestUpdateStatsResetsCurrentTimeoutsOnSuccess(t *testing.T) {
	stats := model.MonitorStats{
		Host:             "www.baidu.com",
		ResolvedIPs:      map[string]int{},
		CurrentTimeouts:  3,
		MaxTimeoutStreak: 3,
	}
	var firstIssueAt time.Time

	updateStats(&stats, model.ProbeResult{
		Time:   time.Date(2026, 3, 26, 10, 5, 0, 0, time.Local),
		Host:   "www.baidu.com",
		IP:     "183.2.172.177",
		Status: model.StatusOK,
		RTT:    6 * time.Millisecond,
	}, &firstIssueAt)

	if stats.CurrentTimeouts != 0 {
		t.Fatalf("expected current timeout streak to reset on success, got %d", stats.CurrentTimeouts)
	}
	if stats.MaxTimeoutStreak != 3 {
		t.Fatalf("expected max timeout streak to be preserved, got %d", stats.MaxTimeoutStreak)
	}
}
