// Package bench measures proxy performance over multiple samples.
package bench

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"golang.org/x/net/proxy"
)

// Stats holds benchmark statistics for a single proxy.
type Stats struct {
	Address    string  `json:"address"`
	Samples    int     `json:"samples"`
	Successful int     `json:"successful"`
	MinMS      int64   `json:"min_ms"`
	MaxMS      int64   `json:"max_ms"`
	AvgMS      int64   `json:"avg_ms"`
	P50MS      int64   `json:"p50_ms"`
	P95MS      int64   `json:"p95_ms"`
	LossRate   float64 `json:"loss_rate"`   // 0.0 – 1.0
	SpeedBps   int64   `json:"speed_bps"`   // bytes/sec of payload download, 0 if not measured
}

// Options configures a benchmark run.
type Options struct {
	Samples     int
	Timeout     time.Duration
	TestURL     string
	PayloadURL  string // optional large URL for throughput measurement
	Concurrency int
}

// DefaultOptions returns sensible benchmark defaults.
func DefaultOptions() Options {
	return Options{
		Samples:  5,
		Timeout:  15 * time.Second,
		TestURL:  "http://www.google.com",
	}
}

// Run executes a benchmark against a single proxy and returns aggregate stats.
func Run(address string, opts Options) Stats {
	stats := Stats{Address: address, Samples: opts.Samples}
	if opts.Samples <= 0 {
		opts.Samples = 5
	}

	client, err := buildClient(address, opts.Timeout)
	if err != nil {
		return stats
	}

	testURL := opts.TestURL
	if testURL == "" {
		testURL = "http://www.google.com"
	}

	latencies := make([]int64, 0, opts.Samples)

	for i := 0; i < opts.Samples; i++ {
		start := time.Now()
		resp, err := client.Get(testURL)
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			continue
		}
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		resp.Body.Close()
		latencies = append(latencies, elapsed)
		stats.Successful++
	}

	if len(latencies) == 0 {
		stats.LossRate = 1.0
		return stats
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	stats.MinMS = latencies[0]
	stats.MaxMS = latencies[len(latencies)-1]
	stats.AvgMS = avg(latencies)
	stats.P50MS = percentile(latencies, 50)
	stats.P95MS = percentile(latencies, 95)
	stats.LossRate = float64(opts.Samples-stats.Successful) / float64(opts.Samples)

	// Optional throughput measurement.
	if opts.PayloadURL != "" {
		stats.SpeedBps = measureSpeed(client, opts.PayloadURL, opts.Timeout)
	}

	return stats
}

// RunMany benchmarks multiple proxies concurrently.
func RunMany(addresses []string, opts Options) []Stats {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 5
	}
	sem := make(chan struct{}, opts.Concurrency)
	results := make([]Stats, len(addresses))
	done := make(chan struct{}, len(addresses))

	for i, addr := range addresses {
		go func(idx int, address string) {
			sem <- struct{}{}
			results[idx] = Run(address, opts)
			<-sem
			done <- struct{}{}
		}(i, addr)
	}
	for range addresses {
		<-done
	}
	return results
}

// buildClient returns an http.Client routed through the proxy at address.
func buildClient(address string, timeout time.Duration) (*http.Client, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("parse proxy URL: %w", err)
	}

	var transport *http.Transport

	switch u.Scheme {
	case "socks5":
		dialer, err := proxy.FromURL(u, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("socks5 dialer: %w", err)
		}
		transport = &http.Transport{Dial: dialer.Dial, DisableKeepAlives: true}
	default:
		// http / https proxy
		transport = &http.Transport{
			Proxy:             http.ProxyURL(u),
			DisableKeepAlives: true,
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

// measureSpeed downloads a URL through the client and returns bytes/sec.
func measureSpeed(client *http.Client, payloadURL string, timeout time.Duration) int64 {
	resp, err := client.Get(payloadURL)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	start := time.Now()
	n, _ := io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start).Seconds()
	if elapsed == 0 {
		return 0
	}
	return int64(float64(n) / elapsed)
}

func avg(vals []int64) int64 {
	var sum int64
	for _, v := range vals {
		sum += v
	}
	return sum / int64(len(vals))
}

func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(p)/100.0*float64(len(sorted)-1) + 0.5)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
