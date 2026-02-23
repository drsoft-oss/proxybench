package checker

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// CheckHTTP validates an HTTP/HTTPS proxy by sending a real request through it.
func CheckHTTP(address string, opts Options) Result {
	result := Result{Address: address, Protocol: ProtocolHTTP}
	if DetectProtocol(address) == ProtocolHTTPS {
		result.Protocol = ProtocolHTTPS
	}

	proxyURL, err := url.Parse(address)
	if err != nil {
		result.Error = fmt.Sprintf("invalid proxy URL: %v", err)
		return result
	}

	transport := &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: opts.Timeout,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
		// Do not follow redirects — we only care about initial response.
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
		result.Error = err.Error()
		return result
	}
	resp.Body.Close()

	result.Alive = true
	result.Latency = elapsed
	return result
}
