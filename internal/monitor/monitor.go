package monitor

import (
	"context"
	"fmt"
	"os"
	"time"

	"netsentry/internal/model"
)

// Prober is implemented by the ICMP client and keeps the monitor package
// decoupled from the Windows-specific probe implementation.
type Prober interface {
	Probe(ctx context.Context, host string, timeout time.Duration) model.ProbeResult
}

type EventLogger interface {
	LogIssue(model.ProbeResult, int) error
	LogRecovery(model.ProbeResult, int, time.Time) error
}

type DailyReporter interface {
	Write(model.MonitorStats) error
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
	OnError   func(err error)
}

// Run drives the monitoring loop, updates counters, prints live status,
// and coordinates event logs plus the daily report file.
func Run(ctx context.Context, cfg Config, prober Prober, logger EventLogger, reporter DailyReporter, hooks Hooks) {
	stats := model.MonitorStats{
		Host:        cfg.Host,
		ResolvedIPs: make(map[string]int),
	}
	newDailyStats := func() model.MonitorStats {
		return model.MonitorStats{Host: cfg.Host, ResolvedIPs: make(map[string]int)}
	}
	dailyStats := newDailyStats()

	var lastStatus model.ProbeStatus
	var hadLastStatus bool
	var firstIssueAt time.Time
	var dailyFirstIssueAt time.Time
	var reportDay string

	checkWrite := func(operation string, err error) {
		if err == nil {
			return
		}
		err = fmt.Errorf("%s: %w", operation, err)
		if hooks.OnError != nil {
			hooks.OnError(err)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	writeReport := func() {
		if dailyStats.Total > 0 {
			checkWrite("write daily report", reporter.Write(dailyStats))
		}
	}
	defer func() {
		writeReport()
		emitStopped(hooks, stats)
	}()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	summaryTicker := time.NewTicker(cfg.SummaryEvery)
	defer summaryTicker.Stop()

	reportTicker := time.NewTicker(cfg.ReportEvery)
	defer reportTicker.Stop()

	runProbe := func() {
		if ctx.Err() != nil {
			return
		}
		result := prober.Probe(ctx, cfg.Host, cfg.Timeout)
		// Stopping during DNS or ICMP is not a failed network sample.
		if ctx.Err() != nil {
			return
		}
		day := result.Time.In(time.Local).Format("20060102")
		newDay := day != reportDay
		if newDay {
			writeReport()
			dailyStats = newDailyStats()
			dailyFirstIssueAt = time.Time{}
			reportDay = day
		}
		updateStats(&stats, result, &firstIssueAt)
		updateStats(&dailyStats, result, &dailyFirstIssueAt)

		if result.Status != model.StatusOK {
			checkWrite("write issue log", logger.LogIssue(result, stats.ConsecutiveIssues))
		}

		recovered := hadLastStatus && lastStatus != model.StatusOK && result.Status == model.StatusOK
		if recovered {
			checkWrite("write recovery log", logger.LogRecovery(result, stats.ConsecutiveIssues, firstIssueAt))
			stats.ConsecutiveIssues = 0
			dailyStats.ConsecutiveIssues = 0
			firstIssueAt = time.Time{}
			dailyFirstIssueAt = time.Time{}
		}
		stateChanged := !hadLastStatus || result.Status != lastStatus
		if newDay || stateChanged {
			writeReport()
		}
		emitProbe(hooks, result, stats)

		lastStatus = result.Status
		hadLastStatus = true
	}

	runProbe()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runProbe()
		case <-summaryTicker.C:
			emitSummary(hooks, stats)
		case <-reportTicker.C:
			writeReport()
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
