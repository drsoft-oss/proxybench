// Package checker validates proxy connectivity and protocol support.
package checker

import (
	"fmt"
	"net"
	"time"
)

// Protocol represents a supported proxy protocol.
type Protocol string

const (
	ProtocolHTTP        Protocol = "http"
	ProtocolHTTPS       Protocol = "https"
	ProtocolSOCKS5      Protocol = "socks5"
	ProtocolShadowsocks Protocol = "ss"
	ProtocolUnknown     Protocol = "unknown"
)

// Result holds the outcome of a proxy check.
type Result struct {
	Address  string        `json:"address"`
	Protocol Protocol      `json:"protocol"`
	Alive    bool          `json:"alive"`
	Latency  time.Duration `json:"latency_ms"`
	Error    string        `json:"error,omitempty"`
}

// LatencyMS returns latency as milliseconds (for serialisation).
func (r Result) LatencyMS() int64 {
	return r.Latency.Milliseconds()
}

// Options configures a check run.
type Options struct {
	Timeout     time.Duration
	TestURL     string // used by HTTP/HTTPS checks
	Concurrency int
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		Timeout:     10 * time.Second,
		TestURL:     "http://www.google.com",
		Concurrency: 10,
	}
}

// DetectProtocol sniffs the scheme prefix; falls back to ProtocolUnknown.
func DetectProtocol(address string) Protocol {
	switch {
	case len(address) >= 7 && address[:7] == "http://":
		return ProtocolHTTP
	case len(address) >= 8 && address[:8] == "https://":
		return ProtocolHTTPS
	case len(address) >= 9 && address[:9] == "socks5://":
		return ProtocolSOCKS5
	case len(address) >= 5 && address[:5] == "ss://":
		return ProtocolShadowsocks
	default:
		return ProtocolUnknown
	}
}

// StripScheme removes the scheme prefix (e.g. "socks5://") from an address.
func StripScheme(address string, proto Protocol) string {
	prefix := string(proto) + "://"
	if len(address) > len(prefix) && address[:len(prefix)] == prefix {
		return address[len(prefix):]
	}
	return address
}

// Check runs a single proxy check, auto-detecting protocol if needed.
func Check(address string, opts Options) Result {
	proto := DetectProtocol(address)

	switch proto {
	case ProtocolHTTP, ProtocolHTTPS:
		return CheckHTTP(address, opts)
	case ProtocolSOCKS5:
		return CheckSOCKS5(address, opts)
	case ProtocolShadowsocks:
		return CheckShadowsocks(address, opts)
	default:
		// Treat bare host:port as SOCKS5 first, fall back to HTTP.
		result := CheckSOCKS5("socks5://"+address, opts)
		if result.Alive {
			return result
		}
		result2 := CheckHTTP("http://"+address, opts)
		if result2.Alive {
			return result2
		}
		return Result{
			Address:  address,
			Protocol: ProtocolUnknown,
			Alive:    false,
			Error:    "protocol auto-detect failed",
		}
	}
}

// CheckMany runs checks concurrently and returns results in input order.
func CheckMany(addresses []string, opts Options) []Result {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 10
	}
	sem := make(chan struct{}, opts.Concurrency)
	results := make([]Result, len(addresses))
	done := make(chan struct{}, len(addresses))

	for i, addr := range addresses {
		go func(idx int, address string) {
			sem <- struct{}{}
			results[idx] = Check(address, opts)
			<-sem
			done <- struct{}{}
		}(i, addr)
	}

	for range addresses {
		<-done
	}
	return results
}

// tcpProbe opens a raw TCP connection and measures latency.
func tcpProbe(host string, timeout time.Duration) (time.Duration, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return 0, fmt.Errorf("tcp dial: %w", err)
	}
	conn.Close()
	return time.Since(start), nil
}
