package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"netsentry/internal/monitor"
)

// AppConfig stores user-editable settings in a JSON file so the GUI can keep
// the monitor target and timing values configurable between launches.
type AppConfig struct {
	Host         string `json:"host"`
	Interval     string `json:"interval"`
	Timeout      string `json:"timeout"`
	LogDir       string `json:"log_dir"`
	ReportDir    string `json:"report_dir"`
	SummaryEvery string `json:"summary_every"`
	ReportEvery  string `json:"report_every"`
}

func Default() AppConfig {
	return AppConfig{
		Host:         "www.baidu.com",
		Interval:     "1s",
		Timeout:      "2s",
		LogDir:       "logs",
		ReportDir:    "reports",
		SummaryEvery: "30s",
		ReportEvery:  "30s",
	}
}

func Load(path string) (AppConfig, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return AppConfig{}, err
	}

	// PowerShell often writes UTF-8 with BOM, so trim it before JSON parsing.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}, err
	}
	cfg.ApplyDefaults()
	return cfg, nil
}

func Save(path string, cfg AppConfig) error {
	cfg.ApplyDefaults()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func (c *AppConfig) ApplyDefaults() {
	defaults := Default()
	if strings.TrimSpace(c.Host) == "" {
		c.Host = defaults.Host
	}
	if strings.TrimSpace(c.Interval) == "" {
		c.Interval = defaults.Interval
	}
	if strings.TrimSpace(c.Timeout) == "" {
		c.Timeout = defaults.Timeout
	}
	if strings.TrimSpace(c.LogDir) == "" {
		c.LogDir = defaults.LogDir
	}
	if strings.TrimSpace(c.ReportDir) == "" {
		c.ReportDir = defaults.ReportDir
	}
	if strings.TrimSpace(c.SummaryEvery) == "" {
		c.SummaryEvery = defaults.SummaryEvery
	}
	if strings.TrimSpace(c.ReportEvery) == "" {
		c.ReportEvery = defaults.ReportEvery
	}

	c.Host = strings.TrimSpace(c.Host)
	c.Interval = strings.TrimSpace(c.Interval)
	c.Timeout = strings.TrimSpace(c.Timeout)
	c.LogDir = strings.TrimSpace(c.LogDir)
	c.ReportDir = strings.TrimSpace(c.ReportDir)
	c.SummaryEvery = strings.TrimSpace(c.SummaryEvery)
	c.ReportEvery = strings.TrimSpace(c.ReportEvery)
}

func (c AppConfig) ToMonitorConfig() (monitor.Config, error) {
	c.ApplyDefaults()

	interval, err := time.ParseDuration(c.Interval)
	if err != nil {
		return monitor.Config{}, fmt.Errorf("invalid interval: %w", err)
	}
	timeout, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return monitor.Config{}, fmt.Errorf("invalid timeout: %w", err)
	}
	summaryEvery, err := time.ParseDuration(c.SummaryEvery)
	if err != nil {
		return monitor.Config{}, fmt.Errorf("invalid summary-every: %w", err)
	}
	reportEvery, err := time.ParseDuration(c.ReportEvery)
	if err != nil {
		return monitor.Config{}, fmt.Errorf("invalid report-every: %w", err)
	}

	if interval < 200*time.Millisecond {
		return monitor.Config{}, fmt.Errorf("interval must be at least 200ms")
	}
	if timeout < 200*time.Millisecond {
		return monitor.Config{}, fmt.Errorf("timeout must be at least 200ms")
	}
	if reportEvery < time.Second {
		return monitor.Config{}, fmt.Errorf("report-every must be at least 1s")
	}

	return monitor.Config{
		Host:         c.Host,
		Interval:     interval,
		Timeout:      timeout,
		SummaryEvery: summaryEvery,
		ReportEvery:  reportEvery,
	}, nil
}
