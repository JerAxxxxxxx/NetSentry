package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTimingValidation(t *testing.T) {
	for _, value := range []string{"0s", "-1s", "500ms", "invalid"} {
		t.Run(value, func(t *testing.T) {
			cfg := Default()
			cfg.SummaryEvery = value
			if _, err := cfg.ToMonitorConfig(); err == nil {
				t.Fatalf("accepted invalid summary interval %q", value)
			}
		})
	}
	cfg := Default()
	cfg.Timeout = "1200h"
	if _, err := cfg.ToMonitorConfig(); err == nil {
		t.Fatal("accepted timeout that would overflow the Windows API")
	}
	cfg.Timeout = (time.Duration(^uint32(0))*time.Millisecond + time.Nanosecond).String()
	if _, err := cfg.ToMonitorConfig(); err == nil {
		t.Fatal("accepted timeout that would overflow when rounded up to milliseconds")
	}
	if _, err := Default().ToMonitorConfig(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigBOMAndReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, []byte(`{"host":"127.0.0.1"}`)...), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil || cfg.Host != "127.0.0.1" {
		t.Fatalf("BOM config: cfg=%+v err=%v", cfg, err)
	}
	for _, host := range []string{"localhost", "192.0.2.1"} {
		cfg.Host = host
		if err := Save(path, cfg); err != nil {
			t.Fatal(err)
		}
		loaded, err := Load(path)
		if err != nil || loaded != cfg {
			t.Fatalf("replacement: cfg=%+v err=%v", loaded, err)
		}
	}
}
