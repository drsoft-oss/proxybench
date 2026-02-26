// Package cmd contains all proxybench CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X github.com/drsoft-oss/proxybench/cmd.version=x.y.z"
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "proxybench",
	Short: "Proxy checker and health monitor",
	Long: `proxybench validates and benchmarks HTTP, SOCKS5, and Shadowsocks proxies.

Features:
  • Protocol auto-detection (http/https/socks5/ss)
  • Speed benchmarks with latency percentiles
  • Geo-location lookup via local IP database
  • JSON and CSV output for pipeline integration
`,
	Version: version,
}

// Execute is the entry point called by main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(benchCmd)
	rootCmd.AddCommand(dbCmd)
}
