package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/romeomihailus/proxybench/internal/checker"
	"github.com/romeomihailus/proxybench/internal/geo"
	"github.com/romeomihailus/proxybench/internal/output"
)

var checkCmd = &cobra.Command{
	Use:   "check [proxy...]",
	Short: "Check one or more proxies for liveness",
	Long: `Check validates proxy liveness and reports protocol, latency, and geo-location.

Proxies can be supplied as arguments or via stdin (one per line).

Examples:
  proxybench check http://1.2.3.4:8080
  proxybench check socks5://10.0.0.1:1080 http://10.0.0.2:3128
  cat proxies.txt | proxybench check --format json
  proxybench check socks5://user:pass@host:1080 --format csv`,
	RunE: runCheck,
}

var (
	checkFormat      string
	checkTimeout     int
	checkTestURL     string
	checkConcurrency int
	checkGeo         bool
	checkDBPath      string
)

func init() {
	checkCmd.Flags().StringVarP(&checkFormat, "format", "f", "table", "output format: table|json|csv")
	checkCmd.Flags().IntVarP(&checkTimeout, "timeout", "t", 10, "per-proxy timeout in seconds")
	checkCmd.Flags().StringVar(&checkTestURL, "test-url", "http://www.google.com", "URL to use for HTTP/SOCKS5 forward checks")
	checkCmd.Flags().IntVarP(&checkConcurrency, "concurrency", "c", 10, "max parallel checks")
	checkCmd.Flags().BoolVar(&checkGeo, "geo", true, "append country info (requires IP database)")
	checkCmd.Flags().StringVar(&checkDBPath, "db", "", "path to ip2country.csv (default: auto-detect)")
}

func runCheck(cmd *cobra.Command, args []string) error {
	addresses := collectAddresses(args)
	if len(addresses) == 0 {
		return fmt.Errorf("no proxy addresses provided; pass them as arguments or via stdin")
	}

	opts := checker.Options{
		Timeout:     time.Duration(checkTimeout) * time.Second,
		TestURL:     checkTestURL,
		Concurrency: checkConcurrency,
	}

	results := checker.CheckMany(addresses, opts)

	var countries []string
	if checkGeo {
		db := geo.DefaultDB
		if checkDBPath != "" {
			if err := db.LoadFile(checkDBPath); err != nil {
				fmt.Fprintf(os.Stderr, "warn: geo DB load failed: %v\n", err)
			}
		} else {
			db.Load() //nolint:errcheck — best effort
		}
		countries = make([]string, len(results))
		for i, r := range results {
			host := extractHost(r.Address)
			if host != "" {
				cc, cn := db.Lookup(host)
				if cc != "--" {
					countries[i] = cc + " " + cn
				}
			}
		}
	}

	return output.WriteCheckResults(os.Stdout, results, countries, output.Format(checkFormat))
}

// collectAddresses merges CLI args with stdin lines.
func collectAddresses(args []string) []string {
	addrs := make([]string, 0, len(args))
	for _, a := range args {
		if s := strings.TrimSpace(a); s != "" {
			addrs = append(addrs, s)
		}
	}

	// If stdin is not a terminal, read from it too.
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				addrs = append(addrs, line)
			}
		}
	}
	return addrs
}

// extractHost returns just the IP/hostname from a proxy address (strips scheme, port, credentials).
func extractHost(address string) string {
	// Strip scheme.
	for _, scheme := range []string{"http://", "https://", "socks5://", "ss://"} {
		address = strings.TrimPrefix(address, scheme)
	}
	// Strip credentials.
	if at := strings.LastIndex(address, "@"); at != -1 {
		address = address[at+1:]
	}
	// Strip port.
	if colon := strings.LastIndex(address, ":"); colon != -1 {
		// Only strip if it looks like host:port (not bare IPv6).
		host := address[:colon]
		if !strings.Contains(host, ":") { // not IPv6
			address = host
		}
	}
	return address
}
