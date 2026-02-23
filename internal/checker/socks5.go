package checker

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// CheckSOCKS5 validates a SOCKS5 proxy.
// It dials through the proxy and performs an HTTP GET to confirm traffic flows.
func CheckSOCKS5(address string, opts Options) Result {
	result := Result{Address: address, Protocol: ProtocolSOCKS5}

	proxyURL, err := url.Parse(address)
	if err != nil {
		result.Error = fmt.Sprintf("invalid socks5 URL: %v", err)
		return result
	}

	// First: fast TCP probe to the proxy itself.
	host := proxyURL.Host
	if _, _, err := net.SplitHostPort(host); err != nil {
		// No port — assume 1080 default.
		host = host + ":1080"
	}

	tcpLatency, err := tcpProbe(host, opts.Timeout)
	if err != nil {
		result.Error = fmt.Sprintf("tcp probe: %v", err)
		return result
	}

	// Second: route an HTTP request through the SOCKS5 proxy.
	dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
	if err != nil {
		result.Error = fmt.Sprintf("socks5 dialer: %v", err)
		return result
	}

	transport := &http.Transport{
		Dial:              dialer.Dial,
		DisableKeepAlives: true,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	testURL := opts.TestURL
	if testURL == "" {
		testURL = "http://www.google.com"
	}

	start := time.Now()
	resp, err := client.Get(testURL)
	elapsed := time.Since(start)

	if err != nil {
		// Proxy is reachable but won't forward — still partially alive.
		result.Alive = false
		result.Latency = tcpLatency
		result.Error = fmt.Sprintf("forward check: %v", err)
		return result
	}
	resp.Body.Close()

	result.Alive = true
	result.Latency = elapsed
	return result
}
