package report

import (
	"os"
	"path/filepath"
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

func TestDailyReporterKeepsSeparateDays(t *testing.T) {
	dir := t.TempDir()
	reporter, err := NewDailyReporter(dir)
	if err != nil {
		t.Fatal(err)
	}
	for day := 26; day <= 27; day++ {
		stats := model.MonitorStats{
			LastProbe: time.Date(2026, 3, day, 23, 0, 0, 0, time.Local),
			Total:     day, Success: day,
		}
		if err := reporter.Write(stats); err != nil {
			t.Fatal(err)
		}
	}
	for _, day := range []string{"26", "27"} {
		path := filepath.Join(dir, "netsentry-report-202603"+day+".txt")
		content, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(content), "Date: 2026-03-"+day) ||
			!strings.Contains(string(content), "Total Probes: "+day) {
			t.Fatalf("wrong daily report %s: %s, %v", path, content, err)
		}
	}
	if filepath.Base(reporter.Path()) != "netsentry-report-20260327.txt" {
		t.Fatal("active report path was not updated")
	}
	if err := reporter.Write(model.MonitorStats{LastProbe: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Write(model.MonitorStats{LastProbe: time.Now(), Total: 2}); err != nil {
		t.Fatal("could not replace existing report:", err)
	}
}
