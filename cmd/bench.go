package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/romeomihailus/proxybench/internal/bench"
	"github.com/romeomihailus/proxybench/internal/output"
)

var benchCmd = &cobra.Command{
	Use:   "bench [proxy...]",
	Short: "Benchmark proxy latency and throughput",
	Long: `Bench runs repeated requests through each proxy and reports latency percentiles.

Examples:
  proxybench bench http://1.2.3.4:8080
  proxybench bench socks5://10.0.0.1:1080 --samples 10 --format json
  cat proxies.txt | proxybench bench --payload-url http://speed.example.com/10mb`,
	RunE: runBench,
}

var (
	benchFormat      string
	benchTimeout     int
	benchSamples     int
	benchTestURL     string
	benchPayloadURL  string
	benchConcurrency int
)

func init() {
	benchCmd.Flags().StringVarP(&benchFormat, "format", "f", "table", "output format: table|json|csv")
	benchCmd.Flags().IntVarP(&benchTimeout, "timeout", "t", 15, "per-request timeout in seconds")
	benchCmd.Flags().IntVarP(&benchSamples, "samples", "n", 5, "number of requests per proxy")
	benchCmd.Flags().StringVar(&benchTestURL, "test-url", "http://www.google.com", "URL to hit for latency measurement")
	benchCmd.Flags().StringVar(&benchPayloadURL, "payload-url", "", "URL of a large file for throughput measurement (optional)")
	benchCmd.Flags().IntVarP(&benchConcurrency, "concurrency", "c", 5, "max parallel proxies under test")
}

func runBench(cmd *cobra.Command, args []string) error {
	addresses := collectAddresses(args)
	if len(addresses) == 0 {
		return fmt.Errorf("no proxy addresses provided")
	}

	opts := bench.Options{
		Samples:     benchSamples,
		Timeout:     time.Duration(benchTimeout) * time.Second,
		TestURL:     benchTestURL,
		PayloadURL:  benchPayloadURL,
		Concurrency: benchConcurrency,
	}

	fmt.Fprintf(os.Stderr, "Benchmarking %d proxies (%d samples each)…\n", len(addresses), benchSamples)
	results := bench.RunMany(addresses, opts)

	return output.WriteBenchResults(os.Stdout, results, output.Format(benchFormat))
}
