package monitor

import (
	"context"
	"errors"
	"strings"
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

type probeFunc func(context.Context, string, time.Duration) model.ProbeResult

func (f probeFunc) Probe(ctx context.Context, host string, timeout time.Duration) model.ProbeResult {
	return f(ctx, host, timeout)
}

type testLogger struct {
	issues        int
	recoveryCount int
	err           error
}

func (l *testLogger) LogIssue(model.ProbeResult, int) error {
	l.issues++
	return l.err
}

func (l *testLogger) LogRecovery(_ model.ProbeResult, streak int, _ time.Time) error {
	l.recoveryCount = streak
	return l.err
}

type testReporter struct {
	reports []model.MonitorStats
	err     error
}

func (r *testReporter) Write(stats model.MonitorStats) error {
	// Snapshot maps as a real file writer would, rather than retaining aliases.
	ips := make(map[string]int)
	for ip, count := range stats.ResolvedIPs {
		ips[ip] = count
	}
	stats.ResolvedIPs = ips
	r.reports = append(r.reports, stats)
	return r.err
}

func runConfig() Config {
	return Config{Host: "localhost", Interval: time.Millisecond, Timeout: time.Second,
		SummaryEvery: time.Hour, ReportEvery: time.Hour}
}

func TestRunDiscardsCanceledProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger, reporter := &testLogger{}, &testReporter{}
	var stopped model.MonitorStats
	Run(ctx, runConfig(), probeFunc(func(context.Context, string, time.Duration) model.ProbeResult {
		cancel()
		return model.ProbeResult{Time: time.Now(), Status: model.StatusError, Detail: "context canceled"}
	}), logger, reporter, Hooks{
		OnProbe:   func(model.ProbeResult, model.MonitorStats) { t.Fatal("emitted a canceled probe") },
		OnStopped: func(stats model.MonitorStats) { stopped = stats },
	})
	if stopped.Total != 0 || logger.issues != 0 || len(reporter.reports) != 0 {
		t.Fatal("cancellation was counted or persisted as packet loss")
	}
}

func TestRunRollsDailyStatsButKeepsSessionAndRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	base := time.Date(2026, 3, 26, 23, 59, 58, 0, time.Local)
	statuses := []model.ProbeStatus{model.StatusTimeout, model.StatusTimeout, model.StatusOK}
	index := 0
	logger, reporter := &testLogger{}, &testReporter{}
	var stopped model.MonitorStats
	Run(ctx, runConfig(), probeFunc(func(context.Context, string, time.Duration) model.ProbeResult {
		result := model.ProbeResult{Time: base.Add(time.Duration(index) * time.Second), Status: statuses[index],
			IP: "127.0.0.1", RTT: 5 * time.Millisecond}
		index++
		return result
	}), logger, reporter, Hooks{
		OnProbe: func(_ model.ProbeResult, stats model.MonitorStats) {
			if stats.Total == len(statuses) {
				if stats.ConsecutiveIssues != 0 {
					t.Fatal("recovery emitted stale consecutive issues")
				}
				cancel()
			}
		},
		OnStopped: func(stats model.MonitorStats) { stopped = stats },
	})
	if stopped.Total != 3 || stopped.Timeout != 2 || stopped.Success != 1 || logger.recoveryCount != 2 {
		t.Fatalf("lost session counters or recovery: %+v, logger=%+v", stopped, logger)
	}
	days := make(map[int]model.MonitorStats)
	for _, stats := range reporter.reports {
		days[stats.LastProbe.Day()] = stats
	}
	if days[26].Total != 2 || days[26].MaxTimeoutStreak != 2 || days[27].Total != 1 ||
		days[27].Success != 1 || days[27].MaxTimeoutStreak != 0 || days[27].LossRate() != 0 {
		t.Fatalf("daily reports mixed samples from different days: %+v", days)
	}
}

func TestRunSurfacesPersistenceFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writeErr := errors.New("disk unavailable")
	logger, reporter := &testLogger{err: writeErr}, &testReporter{err: writeErr}
	var failures []string
	Run(ctx, runConfig(), probeFunc(func(context.Context, string, time.Duration) model.ProbeResult {
		return model.ProbeResult{Time: time.Now(), Status: model.StatusTimeout}
	}), logger, reporter, Hooks{
		OnProbe:   func(model.ProbeResult, model.MonitorStats) { cancel() },
		OnStopped: func(model.MonitorStats) {},
		OnError: func(err error) {
			if !errors.Is(err, writeErr) {
				t.Fatal("lost underlying error:", err)
			}
			failures = append(failures, err.Error())
		},
	})
	output := strings.Join(failures, "\n")
	if !strings.Contains(output, "write issue log") || !strings.Contains(output, "write daily report") {
		t.Fatalf("persistence failures were hidden: %s", output)
	}
}

func TestRepeatedTimeoutsDoNotRewriteReportEveryProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger, reporter := &testLogger{}, &testReporter{}
	Run(ctx, runConfig(), probeFunc(func(context.Context, string, time.Duration) model.ProbeResult {
		return model.ProbeResult{Time: time.Now(), Status: model.StatusTimeout}
	}), logger, reporter, Hooks{
		OnProbe: func(_ model.ProbeResult, stats model.MonitorStats) {
			if stats.Total == 4 {
				cancel()
			}
		},
		OnStopped: func(model.MonitorStats) {},
	})
	if logger.issues != 4 || len(reporter.reports) != 2 || reporter.reports[1].Timeout != 4 {
		t.Fatalf("expected per-issue logs and reports only on transition/exit: issues=%d reports=%+v", logger.issues, reporter.reports)
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
