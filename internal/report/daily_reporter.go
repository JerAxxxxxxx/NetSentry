package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"netsentry/internal/model"
)

// DailyReporter rewrites a compact daily snapshot file. The monitor updates it
// on a timer plus important state transitions to keep disk writes low.
type DailyReporter struct {
	path string
}

func NewDailyReporter(dir string) (*DailyReporter, error) {
	path := filepath.Join(dir, fmt.Sprintf("netsentry-report-%s.txt", time.Now().Format("20060102")))
	file, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	return &DailyReporter{path: path}, nil
}

func (r *DailyReporter) Path() string {
	return r.path
}

func (r *DailyReporter) Write(stats model.MonitorStats) {
	content := buildDailyReport(stats)
	_ = os.WriteFile(r.path, []byte(content), 0o644)
}

func buildDailyReport(stats model.MonitorStats) string {
	now := time.Now().Format("2006-01-02 15:04:05")
	avgRTT := time.Duration(0)
	if stats.Success > 0 {
		avgRTT = stats.RTTSum / time.Duration(stats.Success)
	}

	ips := make([]string, 0, len(stats.ResolvedIPs))
	for ip, count := range stats.ResolvedIPs {
		ips = append(ips, fmt.Sprintf("%s (%d)", ip, count))
	}
	sort.Strings(ips)

	firstProbe := "-"
	if !stats.FirstProbe.IsZero() {
		firstProbe = stats.FirstProbe.Format("2006-01-02 15:04:05")
	}

	lastProbe := "-"
	if !stats.LastProbe.IsZero() {
		lastProbe = stats.LastProbe.Format("2006-01-02 15:04:05")
	}

	lastIssue := "-"
	if !stats.LastIssue.IsZero() {
		lastIssue = stats.LastIssue.Format("2006-01-02 15:04:05")
	}

	resolvedIPs := "-"
	if len(ips) > 0 {
		resolvedIPs = strings.Join(ips, ", ")
	}

	return fmt.Sprintf(
		"NetSentry Daily Report\n\nDate: %s\nUpdated: %s\nHost: %s\nResolved IPs: %s\nFirst Probe: %s\nLast Probe: %s\n\nTotal Probes: %d\nSuccessful Replies: %d\nTimeouts: %d\nUnreachable: %d\nErrors: %d\nLoss Rate: %.2f%%\nAverage RTT: %dms\nMaximum RTT: %dms\nMaximum Consecutive Timeouts: %d\nLast Issue: %s\n",
		time.Now().Format("2006-01-02"),
		now,
		stats.Host,
		resolvedIPs,
		firstProbe,
		lastProbe,
		stats.Total,
		stats.Success,
		stats.Timeout,
		stats.Unreachable,
		stats.Errors,
		stats.LossRate(),
		avgRTT.Milliseconds(),
		stats.MaxRTT.Milliseconds(),
		stats.MaxTimeoutStreak,
		lastIssue,
	)
}
