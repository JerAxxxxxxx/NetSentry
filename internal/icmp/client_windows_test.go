package icmp

import (
	"context"
	"net"
	"testing"
	"time"

	"netsentry/internal/model"
)

func TestWindowsIPv4ByteOrder(t *testing.T) {
	// IPAddr is stored in network byte order, represented as a native DWORD.
	// A symmetric address such as 127.0.0.127 would hide an endian regression.
	if got := intToIPv4(0x0100007f); got != "127.0.0.1" {
		t.Fatalf("incorrect Windows IPAddr conversion: %s", got)
	}
}

func TestLoopbackProbe(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	result := client.Probe(context.Background(), "127.0.0.1", time.Second)
	if result.Status != model.StatusOK || result.IP != "127.0.0.1" {
		t.Fatalf("loopback probe failed: %+v", result)
	}
}

func TestProbeHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// A zero-value client has no handle. Cancellation must be checked before
	// accessing the native API, even when the target is an IP literal.
	result := (&Client{}).Probe(ctx, "127.0.0.1", time.Second)
	if result.Status != model.StatusError || result.Detail != context.Canceled.Error() {
		t.Fatalf("canceled probe was not handled: %+v", result)
	}
}

func TestDNSUsesProbeTimeoutAndCountsTowardTotal(t *testing.T) {
	resolver := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	defer func() { net.DefaultResolver = resolver }()
	budget := 50 * time.Millisecond
	started := time.Now()
	// DNS must finish on its own deadline before the zero-value ICMP client
	// can reach the native API. No external DNS traffic is sent by this test.
	result := (&Client{}).Probe(context.Background(), "netsentry.test", budget)
	if result.Status != model.StatusTimeout || result.TotalLatency < budget || time.Since(started) > 3*time.Second {
		t.Fatalf("DNS exceeded its budget or was omitted from total latency: %+v", result)
	}
}
