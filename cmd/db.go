package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/anonymous-proxies-net/proxybench/internal/geo"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage the IP geo-location database",
}

var dbUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download and install a fresh IP-to-country database",
	Long: `Update downloads the latest free IP-to-country CSV from db-ip.com
and saves it to the proxybench data directory.

The database is used by the 'check' command to resolve proxy IP addresses to
country codes. It is updated monthly by the upstream provider.

Examples:
  proxybench db update
  proxybench db update --dest /etc/proxybench/ip2country.csv
  proxybench db update --timeout 120`,
	RunE: runDBUpdate,
}

var dbInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show information about the currently loaded database",
	RunE:  runDBInfo,
}

var (
	dbUpdateDest    string
	dbUpdateTimeout int
)

func init() {
	dbCmd.AddCommand(dbUpdateCmd)
	dbCmd.AddCommand(dbInfoCmd)

	dbUpdateCmd.Flags().StringVarP(&dbUpdateDest, "dest", "d", "", "destination path (default: auto-detect)")
	dbUpdateCmd.Flags().IntVarP(&dbUpdateTimeout, "timeout", "t", 120, "download timeout in seconds")
}

func runDBUpdate(cmd *cobra.Command, args []string) error {
	opts := geo.UpdateOptions{
		DestPath: dbUpdateDest,
		Timeout:  time.Duration(dbUpdateTimeout) * time.Second,
		Progress: func(msg string) {
			fmt.Fprintln(os.Stderr, msg)
		},
	}

	if err := geo.Update(opts); err != nil {
		return fmt.Errorf("db update failed: %w", err)
	}

	// Verify the file loads cleanly.
	dest := dbUpdateDest
	if dest == "" {
		dest = geo.DefaultDBPath()
	}
	fmt.Fprintf(os.Stderr, "Verifying database…\n")
	db := &geo.DB{}
	if err := db.LoadFile(dest); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "✓ Database loaded successfully (%d entries)\n", db.Count())
	return nil
}

func runDBInfo(cmd *cobra.Command, args []string) error {
	path := geo.DefaultDBPath()
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "No database found at %s\n", path)
			fmt.Fprintf(os.Stderr, "Run `proxybench db update` to download it.\n")
			return nil
		}
		return err
	}

	fmt.Printf("Path:     %s\n", path)
	fmt.Printf("Size:     %.1f MB\n", float64(info.Size())/(1<<20))
	fmt.Printf("Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

	db := &geo.DB{}
	if err := db.LoadFile(path); err != nil {
		fmt.Printf("Status:   ERROR - %v\n", err)
	} else {
		fmt.Printf("Entries:  %d\n", db.Count())
		fmt.Printf("Status:   OK\n")
	}
	return nil
}
