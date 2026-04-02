package report

import (
	"strings"
	"testing"
	"time"

	"netsentry/internal/model"
)

func TestBuildDailyReportContainsKeyMetrics(t *testing.T) {
	stats := model.MonitorStats{
		Host:             "www.baidu.com",
		Total:            10,
		Success:          8,
		Timeout:          2,
		MaxTimeoutStreak: 2,
		RTTSum:           80 * time.Millisecond,
		MaxRTT:           16 * time.Millisecond,
		FirstProbe:       time.Date(2026, 3, 26, 9, 0, 0, 0, time.Local),
		LastProbe:        time.Date(2026, 3, 26, 9, 10, 0, 0, time.Local),
		LastIssue:        time.Date(2026, 3, 26, 9, 9, 0, 0, time.Local),
		ResolvedIPs: map[string]int{
			"183.2.172.177": 8,
		},
	}

	report := buildDailyReport(stats)

	for _, part := range []string{
		"Host: www.baidu.com",
		"Loss Rate: 20.00%",
		"Average RTT: 10ms",
		"Maximum RTT: 16ms",
		"Maximum Consecutive Timeouts: 2",
		"183.2.172.177 (8)",
	} {
		if !strings.Contains(report, part) {
			t.Fatalf("expected report to contain %q\nreport:\n%s", part, report)
		}
	}
}
