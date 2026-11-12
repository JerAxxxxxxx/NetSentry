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

	if err := logger.LogIssue(model.ProbeResult{
		Time:         issueTime,
		Host:         "www.baidu.com",
		IP:           "183.2.172.177",
		Status:       model.StatusTimeout,
		RTT:          0,
		TotalLatency: 2 * time.Second,
		Detail:       "request timed out",
		RawSummary:   "request timed out",
	}, 2); err != nil {
		t.Fatal(err)
	}

	if err := logger.LogRecovery(model.ProbeResult{
		Time:         recoveryTime,
		Host:         "www.baidu.com",
		IP:           "183.2.172.177",
		Status:       model.StatusOK,
		RTT:          7 * time.Millisecond,
		TotalLatency: 8 * time.Millisecond,
	}, 2, issueTime); err != nil {
		t.Fatal(err)
	}

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

func TestEventLoggerRotatesAtMidnight(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewEventLogger(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()
	before := time.Date(2026, 3, 26, 23, 59, 59, 0, time.Local)
	if err := logger.LogIssue(model.ProbeResult{Time: before, Status: model.StatusTimeout}, 1); err != nil {
		t.Fatal(err)
	}
	if err := logger.LogRecovery(model.ProbeResult{Time: before.Add(time.Second), Status: model.StatusOK}, 1, before); err != nil {
		t.Fatal(err)
	}
	for name, expected := range map[string]string{
		"netsentry-20260326.log": "ISSUE",
		"netsentry-20260327.log": "RECOVERED",
	} {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !strings.Contains(string(content), expected) {
			t.Fatalf("%s: %q, %v", name, content, err)
		}
	}
}
