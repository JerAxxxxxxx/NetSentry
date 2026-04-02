package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"netsentry/internal/model"
)

// EventLogger persists issue and recovery events separately from the daily
// aggregate report so troubleshooting always has raw incident records.
type EventLogger struct {
	file *os.File
	path string
}

func NewEventLogger(dir string) (*EventLogger, error) {
	fileName := fmt.Sprintf("netsentry-%s.log", time.Now().Format("20060102"))
	path := filepath.Join(dir, fileName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &EventLogger{file: file, path: path}, nil
}

func (l *EventLogger) Path() string {
	return l.path
}

func (l *EventLogger) Close() error {
	return l.file.Close()
}

func (l *EventLogger) LogIssue(result model.ProbeResult, streak int) {
	_, _ = fmt.Fprintf(
		l.file,
		"[%s] ISSUE host=%s ip=%s status=%s streak=%d rtt=%dms total=%dms detail=%s raw=%s\n",
		result.Time.Format("2006-01-02 15:04:05"),
		result.Host,
		result.IP,
		result.Status,
		streak,
		result.RTT.Milliseconds(),
		result.TotalLatency.Milliseconds(),
		sanitize(result.Detail),
		sanitize(result.RawSummary),
	)
}

func (l *EventLogger) LogRecovery(result model.ProbeResult, streak int, firstIssueAt time.Time) {
	duration := result.Time.Sub(firstIssueAt)
	if firstIssueAt.IsZero() {
		duration = 0
	}

	_, _ = fmt.Fprintf(
		l.file,
		"[%s] RECOVERED host=%s ip=%s previous-streak=%d outage-duration=%s rtt=%dms total=%dms\n",
		result.Time.Format("2006-01-02 15:04:05"),
		result.Host,
		result.IP,
		streak,
		duration.Round(time.Millisecond),
		result.RTT.Milliseconds(),
		result.TotalLatency.Milliseconds(),
	)
}

func sanitize(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}
