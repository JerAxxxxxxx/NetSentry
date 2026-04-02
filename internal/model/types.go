package model

import "time"

// ProbeStatus describes the outcome of a single network probe.
type ProbeStatus string

const (
	StatusOK          ProbeStatus = "ok"
	StatusTimeout     ProbeStatus = "timeout"
	StatusUnreachable ProbeStatus = "unreachable"
	StatusError       ProbeStatus = "error"
)

// ProbeResult captures one probe result, including both network RTT and the
// local end-to-end probe cost seen by the application.
type ProbeResult struct {
	Time         time.Time
	Host         string
	IP           string
	Status       ProbeStatus
	RTT          time.Duration
	TotalLatency time.Duration
	Detail       string
	RawSummary   string
}

// MonitorStats stores the rolling counters used for console output, logs,
// and daily reports.
type MonitorStats struct {
	Total             int
	Success           int
	Timeout           int
	Unreachable       int
	Errors            int
	ConsecutiveIssues int
	CurrentTimeouts   int
	MaxTimeoutStreak  int
	LastIssue         time.Time
	FirstProbe        time.Time
	LastProbe         time.Time
	RTTSum            time.Duration
	MaxRTT            time.Duration
	Host              string
	ResolvedIPs       map[string]int
}

// LossRate returns the current failure percentage across all probes.
func (s MonitorStats) LossRate() float64 {
	if s.Total == 0 {
		return 0
	}

	failures := s.Timeout + s.Unreachable + s.Errors
	return float64(failures) * 100 / float64(s.Total)
}
