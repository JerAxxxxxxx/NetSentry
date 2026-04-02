package monitor

import (
	"context"
	"fmt"
	"time"

	"netsentry/internal/logging"
	"netsentry/internal/model"
	"netsentry/internal/report"
)

// Prober is implemented by the ICMP client and keeps the monitor package
// decoupled from the Windows-specific probe implementation.
type Prober interface {
	Probe(ctx context.Context, host string, timeout time.Duration) model.ProbeResult
}

type Config struct {
	Host         string
	Interval     time.Duration
	Timeout      time.Duration
	SummaryEvery time.Duration
	ReportEvery  time.Duration
}

// Hooks lets callers observe monitor activity without forcing the monitor
// package to know whether it is running in a CLI or GUI environment.
type Hooks struct {
	OnProbe   func(result model.ProbeResult, stats model.MonitorStats)
	OnSummary func(stats model.MonitorStats)
	OnStopped func(stats model.MonitorStats)
}

// Run drives the monitoring loop, updates counters, prints live status,
// and coordinates event logs plus the daily report file.
func Run(ctx context.Context, cfg Config, prober Prober, logger *logging.EventLogger, reporter *report.DailyReporter, hooks Hooks) {
	stats := model.MonitorStats{
		Host:        cfg.Host,
		ResolvedIPs: make(map[string]int),
	}

	var lastStatus model.ProbeStatus
	var hadLastStatus bool
	var firstIssueAt time.Time

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	summaryTicker := time.NewTicker(cfg.SummaryEvery)
	defer summaryTicker.Stop()

	reportTicker := time.NewTicker(cfg.ReportEvery)
	defer reportTicker.Stop()

	runProbe := func() {
		result := prober.Probe(ctx, cfg.Host, cfg.Timeout)
		updateStats(&stats, result, &firstIssueAt)
		emitProbe(hooks, result, stats)

		if result.Status != model.StatusOK {
			logger.LogIssue(result, stats.ConsecutiveIssues)
			reporter.Write(stats)
		}

		if hadLastStatus && lastStatus != model.StatusOK && result.Status == model.StatusOK {
			logger.LogRecovery(result, stats.ConsecutiveIssues, firstIssueAt)
			reporter.Write(stats)
			stats.ConsecutiveIssues = 0
			firstIssueAt = time.Time{}
		}

		lastStatus = result.Status
		hadLastStatus = true
	}

	runProbe()

	for {
		select {
		case <-ctx.Done():
			reporter.Write(stats)
			emitStopped(hooks, stats)
			return
		case <-ticker.C:
			runProbe()
		case <-summaryTicker.C:
			emitSummary(hooks, stats)
		case <-reportTicker.C:
			reporter.Write(stats)
		}
	}
}

func updateStats(stats *model.MonitorStats, result model.ProbeResult, firstIssueAt *time.Time) {
	if stats.FirstProbe.IsZero() {
		stats.FirstProbe = result.Time
	}
	stats.LastProbe = result.Time
	stats.Total++

	if result.IP != "" {
		stats.ResolvedIPs[result.IP]++
	}

	switch result.Status {
	case model.StatusOK:
		stats.Success++
		stats.CurrentTimeouts = 0
		stats.RTTSum += result.RTT
		if result.RTT > stats.MaxRTT {
			stats.MaxRTT = result.RTT
		}
	case model.StatusTimeout:
		stats.Timeout++
		stats.ConsecutiveIssues++
		stats.CurrentTimeouts++
		if stats.CurrentTimeouts > stats.MaxTimeoutStreak {
			stats.MaxTimeoutStreak = stats.CurrentTimeouts
		}
		stats.LastIssue = result.Time
		if firstIssueAt.IsZero() {
			*firstIssueAt = result.Time
		}
	case model.StatusUnreachable:
		stats.Unreachable++
		stats.ConsecutiveIssues++
		stats.CurrentTimeouts = 0
		stats.LastIssue = result.Time
		if firstIssueAt.IsZero() {
			*firstIssueAt = result.Time
		}
	default:
		stats.Errors++
		stats.ConsecutiveIssues++
		stats.CurrentTimeouts = 0
		stats.LastIssue = result.Time
		if firstIssueAt.IsZero() {
			*firstIssueAt = result.Time
		}
	}
}

func printLine(result model.ProbeResult, stats model.MonitorStats) {
	ts := result.Time.Format("2006-01-02 15:04:05")
	switch result.Status {
	case model.StatusOK:
		fmt.Printf("[%s] OK ip=%s rtt=%dms total=%dms loss-rate=%.2f%%\n",
			ts,
			result.IP,
			result.RTT.Milliseconds(),
			result.TotalLatency.Milliseconds(),
			stats.LossRate(),
		)
	case model.StatusTimeout:
		fmt.Printf("[%s] TIMEOUT total=%dms loss-rate=%.2f%%\n",
			ts,
			result.TotalLatency.Milliseconds(),
			stats.LossRate(),
		)
	case model.StatusUnreachable:
		fmt.Printf("[%s] UNREACHABLE detail=%s total=%dms loss-rate=%.2f%%\n",
			ts,
			result.Detail,
			result.TotalLatency.Milliseconds(),
			stats.LossRate(),
		)
	default:
		fmt.Printf("[%s] ERROR detail=%s total=%dms loss-rate=%.2f%%\n",
			ts,
			result.Detail,
			result.TotalLatency.Milliseconds(),
			stats.LossRate(),
		)
	}
}

func emitProbe(hooks Hooks, result model.ProbeResult, stats model.MonitorStats) {
	if hooks.OnProbe != nil {
		hooks.OnProbe(result, stats)
		return
	}
	printLine(result, stats)
}

func emitSummary(hooks Hooks, stats model.MonitorStats) {
	if hooks.OnSummary != nil {
		hooks.OnSummary(stats)
		return
	}

	fmt.Printf("[summary] total=%d success=%d timeout=%d unreachable=%d error=%d loss-rate=%.2f%%\n",
		stats.Total,
		stats.Success,
		stats.Timeout,
		stats.Unreachable,
		stats.Errors,
		stats.LossRate(),
	)
}

func emitStopped(hooks Hooks, stats model.MonitorStats) {
	if hooks.OnStopped != nil {
		hooks.OnStopped(stats)
		return
	}

	fmt.Println()
	fmt.Printf("Stopped. total=%d success=%d timeout=%d unreachable=%d error=%d loss-rate=%.2f%%\n",
		stats.Total,
		stats.Success,
		stats.Timeout,
		stats.Unreachable,
		stats.Errors,
		stats.LossRate(),
	)
}
