package main

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"netsentry/internal/config"
)

func TestConcurrentStartsCreateOneSession(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	app.configPath = filepath.Join(dir, "config.json")
	cfg := config.Default()
	cfg.Host = "127.0.0.1"
	cfg.Timeout = "200ms"
	cfg.LogDir, cfg.ReportDir = filepath.Join(dir, "logs"), filepath.Join(dir, "reports")
	defer app.shutdown(context.Background())
	const workers = 8
	gate := make(chan struct{})
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			results <- app.StartMonitoring(cfg)
		}()
	}
	close(gate)
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one monitor session, got %d", successes)
	}
}

func TestShutdownWaitsForSessionCleanup(t *testing.T) {
	app := NewApp()
	canceled := make(chan struct{})
	session := &monitorSession{cancel: func() { close(canceled) }, done: make(chan struct{})}
	app.session = session
	returned := make(chan struct{})
	go func() {
		app.shutdown(context.Background())
		close(returned)
	}()
	<-canceled
	select {
	case <-returned:
		t.Fatal("shutdown returned before the session flushed its report")
	default:
	}
	close(session.done)
	select {
	case <-returned:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not return after session cleanup")
	}
	if err := app.StartMonitoring(config.Default()); err == nil {
		t.Fatal("started a monitor after shutdown")
	}
}

func TestFailedStartReleasesSession(t *testing.T) {
	app := NewApp()
	cfg := config.Default()
	cfg.SummaryEvery = "0s"
	if err := app.StartMonitoring(cfg); err == nil {
		t.Fatal("accepted invalid config")
	}
	if app.IsRunning() {
		t.Fatal("failed initialization left a stale session")
	}
}
