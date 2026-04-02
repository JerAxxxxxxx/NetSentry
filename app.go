package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"netsentry/internal/config"
	"netsentry/internal/icmp"
	"netsentry/internal/logging"
	"netsentry/internal/model"
	"netsentry/internal/monitor"
	"netsentry/internal/report"
)

type monitorSession struct {
	cancel   context.CancelFunc
	client   *icmp.Client
	logger   *logging.EventLogger
	reporter *report.DailyReporter
}

type ProbeEvent struct {
	Time         string  `json:"time"`
	Host         string  `json:"host"`
	IP           string  `json:"ip"`
	Status       string  `json:"status"`
	RTTMs        int64   `json:"rttMs"`
	TotalMs      int64   `json:"totalMs"`
	LossRate     float64 `json:"lossRate"`
	Detail       string  `json:"detail"`
	SummaryLine  string  `json:"summaryLine"`
	Total        int     `json:"total"`
	Success      int     `json:"success"`
	Timeout      int     `json:"timeout"`
	Unreachable  int     `json:"unreachable"`
	Errors       int     `json:"errors"`
	MaxTimeouts  int     `json:"maxTimeouts"`
	CurrentIPSet string  `json:"currentIPSet"`
}

type SummaryEvent struct {
	Total       int     `json:"total"`
	Success     int     `json:"success"`
	Timeout     int     `json:"timeout"`
	Unreachable int     `json:"unreachable"`
	Errors      int     `json:"errors"`
	LossRate    float64 `json:"lossRate"`
	MaxTimeouts int     `json:"maxTimeouts"`
	SummaryLine string  `json:"summaryLine"`
}

// App is the Wails backend that coordinates configuration, monitoring, and
// event streaming to the desktop frontend.
type App struct {
	ctx        context.Context
	configPath string
	mu         sync.Mutex
	session    *monitorSession
}

func NewApp() *App {
	return &App{configPath: "netsentry.json"}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(context.Context) {
	a.StopMonitoring()
}

func (a *App) GetConfig() (config.AppConfig, error) {
	cfg, err := config.Load(a.configPath)
	if err != nil {
		return config.AppConfig{}, err
	}
	cfg.ApplyDefaults()
	return cfg, nil
}

func (a *App) SaveConfig(cfg config.AppConfig) error {
	cfg.ApplyDefaults()
	if _, err := cfg.ToMonitorConfig(); err != nil {
		return err
	}
	return config.Save(a.configPath, cfg)
}

func (a *App) StartMonitoring(cfg config.AppConfig) error {
	a.mu.Lock()
	if a.session != nil {
		a.mu.Unlock()
		return fmt.Errorf("monitor is already running")
	}
	a.mu.Unlock()

	cfg.ApplyDefaults()
	monitorCfg, err := cfg.ToMonitorConfig()
	if err != nil {
		return err
	}
	if err := config.Save(a.configPath, cfg); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.LogDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.ReportDir, 0o755); err != nil {
		return err
	}

	client, err := icmp.NewClient()
	if err != nil {
		return err
	}
	logger, err := logging.NewEventLogger(cfg.LogDir)
	if err != nil {
		_ = client.Close()
		return err
	}
	reporter, err := report.NewDailyReporter(cfg.ReportDir)
	if err != nil {
		_ = logger.Close()
		_ = client.Close()
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &monitorSession{cancel: cancel, client: client, logger: logger, reporter: reporter}

	a.mu.Lock()
	a.session = session
	a.mu.Unlock()

	wruntime.EventsEmit(a.ctx, "monitor:started", map[string]any{
		"reportPath": reporter.Path(),
	})

	go func() {
		defer logger.Close()
		defer client.Close()
		monitor.Run(ctx, monitorCfg, client, logger, reporter, monitor.Hooks{
			OnProbe: func(result model.ProbeResult, stats model.MonitorStats) {
				wruntime.EventsEmit(a.ctx, "monitor:probe", toProbeEvent(result, stats))
			},
			OnSummary: func(stats model.MonitorStats) {
				wruntime.EventsEmit(a.ctx, "monitor:summary", toSummaryEvent(stats))
			},
			OnStopped: func(stats model.MonitorStats) {
				a.mu.Lock()
				a.session = nil
				a.mu.Unlock()
				wruntime.EventsEmit(a.ctx, "monitor:stopped", toSummaryEvent(stats))
			},
		})
	}()

	return nil
}

func (a *App) StopMonitoring() {
	a.mu.Lock()
	session := a.session
	a.mu.Unlock()
	if session != nil && session.cancel != nil {
		session.cancel()
	}
}

func (a *App) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.session != nil
}

func (a *App) OpenPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	resolved, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(resolved, 0o755); err != nil {
		return err
	}

	return exec.Command("explorer", resolved).Start()
}

func toProbeEvent(result model.ProbeResult, stats model.MonitorStats) ProbeEvent {
	return ProbeEvent{
		Time:         result.Time.Format("2006-01-02 15:04:05"),
		Host:         result.Host,
		IP:           result.IP,
		Status:       string(result.Status),
		RTTMs:        result.RTT.Milliseconds(),
		TotalMs:      result.TotalLatency.Milliseconds(),
		LossRate:     stats.LossRate(),
		Detail:       result.Detail,
		SummaryLine:  formatProbeLine(result, stats),
		Total:        stats.Total,
		Success:      stats.Success,
		Timeout:      stats.Timeout,
		Unreachable:  stats.Unreachable,
		Errors:       stats.Errors,
		MaxTimeouts:  stats.MaxTimeoutStreak,
		CurrentIPSet: result.IP,
	}
}

func toSummaryEvent(stats model.MonitorStats) SummaryEvent {
	return SummaryEvent{
		Total:       stats.Total,
		Success:     stats.Success,
		Timeout:     stats.Timeout,
		Unreachable: stats.Unreachable,
		Errors:      stats.Errors,
		LossRate:    stats.LossRate(),
		MaxTimeouts: stats.MaxTimeoutStreak,
		SummaryLine: fmt.Sprintf("[summary] total=%d success=%d timeout=%d unreachable=%d error=%d loss-rate=%.2f%%", stats.Total, stats.Success, stats.Timeout, stats.Unreachable, stats.Errors, stats.LossRate()),
	}
}

func formatProbeLine(result model.ProbeResult, stats model.MonitorStats) string {
	ts := result.Time.Format("2006-01-02 15:04:05")
	switch result.Status {
	case model.StatusOK:
		return fmt.Sprintf("[%s] OK ip=%s rtt=%dms total=%dms loss-rate=%.2f%%", ts, result.IP, result.RTT.Milliseconds(), result.TotalLatency.Milliseconds(), stats.LossRate())
	case model.StatusTimeout:
		return fmt.Sprintf("[%s] TIMEOUT total=%dms loss-rate=%.2f%%", ts, result.TotalLatency.Milliseconds(), stats.LossRate())
	case model.StatusUnreachable:
		return fmt.Sprintf("[%s] UNREACHABLE detail=%s total=%dms loss-rate=%.2f%%", ts, result.Detail, result.TotalLatency.Milliseconds(), stats.LossRate())
	default:
		return fmt.Sprintf("[%s] ERROR detail=%s total=%dms loss-rate=%.2f%%", ts, result.Detail, result.TotalLatency.Milliseconds(), stats.LossRate())
	}
}
