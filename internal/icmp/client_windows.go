package icmp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"
	"unsafe"

	"netsentry/internal/model"
)

const (
	ipSuccess             = 0
	ipDestNetUnreachable  = 11002
	ipDestHostUnreachable = 11003
	ipReqTimedOut         = 11010
)

type ipOptionInformation struct {
	TTL         byte
	TOS         byte
	Flags       byte
	OptionsSize byte
	OptionsData uintptr
}

type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       ipOptionInformation
}

// Client wraps the Windows ICMP API. Using the native API avoids repeatedly
// spawning ping.exe and keeps the monitor lightweight for long-running use.
type Client struct {
	handle          syscall.Handle
	procSendEcho    *syscall.LazyProc
	procCloseHandle *syscall.LazyProc
}

// NewClient initializes the Windows ICMP handle once for the lifetime of the
// monitor process.
func NewClient() (*Client, error) {
	dll := syscall.NewLazyDLL("iphlpapi.dll")
	procCreateFile := dll.NewProc("IcmpCreateFile")
	procSendEcho := dll.NewProc("IcmpSendEcho")
	procCloseHandle := dll.NewProc("IcmpCloseHandle")

	r1, _, callErr := procCreateFile.Call()
	if r1 == 0 || r1 == uintptr(syscall.InvalidHandle) {
		if callErr != syscall.Errno(0) {
			return nil, callErr
		}
		return nil, errors.New("IcmpCreateFile returned invalid handle")
	}

	return &Client{
		handle:          syscall.Handle(r1),
		procSendEcho:    procSendEcho,
		procCloseHandle: procCloseHandle,
	}, nil
}

func (c *Client) Close() error {
	if c.handle == 0 {
		return nil
	}

	r1, _, callErr := c.procCloseHandle.Call(uintptr(c.handle))
	c.handle = 0
	if r1 == 0 && callErr != syscall.Errno(0) {
		return callErr
	}
	return nil
}

// Probe performs one IPv4 ICMP echo request and returns both RTT and the
// application-observed total latency for the entire system call.
func (c *Client) Probe(ctx context.Context, host string, timeout time.Duration) model.ProbeResult {
	now := time.Now()
	ip, err := resolveIPv4(ctx, host)
	if err != nil {
		return model.ProbeResult{
			Time:         now,
			Host:         host,
			Status:       model.StatusError,
			TotalLatency: 0,
			Detail:       fmt.Sprintf("resolve failed: %v", err),
			RawSummary:   "dns lookup failed",
		}
	}

	payload := []byte("NetSentry")
	replyBuffer := make([]byte, int(unsafe.Sizeof(icmpEchoReply{}))+len(payload)+8)
	ipAddr := binary.LittleEndian.Uint32(ip)

	start := time.Now()
	r1, _, callErr := c.procSendEcho.Call(
		uintptr(c.handle),
		uintptr(ipAddr),
		uintptr(unsafe.Pointer(&payload[0])),
		uintptr(len(payload)),
		0,
		uintptr(unsafe.Pointer(&replyBuffer[0])),
		uintptr(len(replyBuffer)),
		uintptr(timeout.Milliseconds()),
	)
	totalLatency := time.Since(start)

	if ctx.Err() != nil {
		return model.ProbeResult{
			Time:         now,
			Host:         host,
			IP:           ip.String(),
			Status:       model.StatusError,
			TotalLatency: totalLatency,
			Detail:       ctx.Err().Error(),
			RawSummary:   "probe canceled",
		}
	}

	if r1 == 0 {
		status, detail := classifyStatus(ipReqTimedOut)
		if errno, ok := callErr.(syscall.Errno); ok && errno != 0 {
			status, detail = classifyStatus(uint32(errno))
		}

		return model.ProbeResult{
			Time:         now,
			Host:         host,
			IP:           ip.String(),
			Status:       status,
			TotalLatency: totalLatency,
			Detail:       detail,
			RawSummary:   detail,
		}
	}

	reply := (*icmpEchoReply)(unsafe.Pointer(&replyBuffer[0]))
	status, detail := classifyStatus(reply.Status)

	return model.ProbeResult{
		Time:         now,
		Host:         host,
		IP:           intToIPv4(reply.Address),
		Status:       status,
		RTT:          time.Duration(reply.RoundTripTime) * time.Millisecond,
		TotalLatency: totalLatency,
		Detail:       detail,
		RawSummary:   fmt.Sprintf("reply ip=%s status=%d rtt=%dms", intToIPv4(reply.Address), reply.Status, reply.RoundTripTime),
	}
}

func resolveIPv4(ctx context.Context, host string) (net.IP, error) {
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	for _, addr := range ips {
		if ipv4 := addr.IP.To4(); ipv4 != nil {
			return ipv4, nil
		}
	}

	return nil, errors.New("no IPv4 address found")
}

func classifyStatus(code uint32) (model.ProbeStatus, string) {
	switch code {
	case ipSuccess:
		return model.StatusOK, "reply received"
	case ipReqTimedOut:
		return model.StatusTimeout, "request timed out"
	case ipDestNetUnreachable:
		return model.StatusUnreachable, "destination network unreachable"
	case ipDestHostUnreachable:
		return model.StatusUnreachable, "destination host unreachable"
	default:
		return model.StatusError, fmt.Sprintf("icmp status=%d", code)
	}
}

func intToIPv4(value uint32) string {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, value)
	return net.IP(bytes).String()
}
