package checker

import (
	"testing"
	"time"
)

func TestDetectProtocol(t *testing.T) {
	cases := []struct {
		addr string
		want Protocol
	}{
		{"http://1.2.3.4:8080", ProtocolHTTP},
		{"https://1.2.3.4:8080", ProtocolHTTPS},
		{"socks5://1.2.3.4:1080", ProtocolSOCKS5},
		{"ss://abc@1.2.3.4:8388", ProtocolShadowsocks},
		{"1.2.3.4:8080", ProtocolUnknown},
		{"", ProtocolUnknown},
	}
	for _, c := range cases {
		got := DetectProtocol(c.addr)
		if got != c.want {
			t.Errorf("DetectProtocol(%q) = %q, want %q", c.addr, got, c.want)
		}
	}
}

func TestStripScheme(t *testing.T) {
	cases := []struct {
		addr  string
		proto Protocol
		want  string
	}{
		{"socks5://1.2.3.4:1080", ProtocolSOCKS5, "1.2.3.4:1080"},
		{"http://1.2.3.4:8080", ProtocolHTTP, "1.2.3.4:8080"},
		{"1.2.3.4:8080", ProtocolHTTP, "1.2.3.4:8080"}, // no prefix to strip
	}
	for _, c := range cases {
		got := StripScheme(c.addr, c.proto)
		if got != c.want {
			t.Errorf("StripScheme(%q, %q) = %q, want %q", c.addr, c.proto, got, c.want)
		}
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.Timeout != 10*time.Second {
		t.Errorf("default timeout = %v, want 10s", opts.Timeout)
	}
	if opts.Concurrency != 10 {
		t.Errorf("default concurrency = %d, want 10", opts.Concurrency)
	}
}

func TestCheckMany_emptyInput(t *testing.T) {
	results := CheckMany(nil, DefaultOptions())
	if len(results) != 0 {
		t.Errorf("expected empty results for nil input, got %d", len(results))
	}
}

func TestResultLatencyMS(t *testing.T) {
	r := Result{Latency: 150 * time.Millisecond}
	if r.LatencyMS() != 150 {
		t.Errorf("LatencyMS() = %d, want 150", r.LatencyMS())
	}
}
