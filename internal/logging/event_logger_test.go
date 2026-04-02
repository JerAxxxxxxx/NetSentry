package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"netsentry/internal/model"
)

func TestEventLoggerWritesIssueAndRecovery(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewEventLogger(dir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	issueTime := time.Date(2026, 3, 26, 12, 0, 0, 0, time.Local)
	recoveryTime := issueTime.Add(2 * time.Second)

	logger.LogIssue(model.ProbeResult{
		Time:         issueTime,
		Host:         "www.baidu.com",
		IP:           "183.2.172.177",
		Status:       model.StatusTimeout,
		RTT:          0,
		TotalLatency: 2 * time.Second,
		Detail:       "request timed out",
		RawSummary:   "request timed out",
	}, 2)

	logger.LogRecovery(model.ProbeResult{
		Time:         recoveryTime,
		Host:         "www.baidu.com",
		IP:           "183.2.172.177",
		Status:       model.StatusOK,
		RTT:          7 * time.Millisecond,
		TotalLatency: 8 * time.Millisecond,
	}, 2, issueTime)

	contentBytes, err := os.ReadFile(filepath.Clean(logger.Path()))
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	content := string(contentBytes)

	for _, part := range []string{
		"ISSUE host=www.baidu.com ip=183.2.172.177 status=timeout streak=2",
		"RECOVERED host=www.baidu.com ip=183.2.172.177 previous-streak=2",
		"outage-duration=2s",
	} {
		if !strings.Contains(content, part) {
			t.Fatalf("expected log output to contain %q\ncontent:\n%s", part, content)
		}
	}
}

func TestSanitizeRemovesLineBreaks(t *testing.T) {
	got := sanitize("line1\r\nline2\nline3")
	if got != "line1  line2 line3" {
		t.Fatalf("unexpected sanitize result: %q", got)
	}
}
