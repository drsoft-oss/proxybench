// Package output formats proxy check and benchmark results as JSON or CSV.
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/romeomihailus/proxybench/internal/bench"
	"github.com/romeomihailus/proxybench/internal/checker"
)

// Format selects the output format.
type Format string

const (
	FormatJSON  Format = "json"
	FormatCSV   Format = "csv"
	FormatTable Format = "table"
)

// ---- Check results ----------------------------------------------------------

// checkRow is the serialisable form of a checker.Result (latency as int64).
type checkRow struct {
	Address  string `json:"address"`
	Protocol string `json:"protocol"`
	Alive    bool   `json:"alive"`
	LatencyMS int64 `json:"latency_ms"`
	Country  string `json:"country,omitempty"`
	Error    string `json:"error,omitempty"`
}

func toCheckRow(r checker.Result, country string) checkRow {
	return checkRow{
		Address:   r.Address,
		Protocol:  string(r.Protocol),
		Alive:     r.Alive,
		LatencyMS: r.LatencyMS(),
		Country:   country,
		Error:     r.Error,
	}
}

// WriteCheckResults writes check results in the requested format.
func WriteCheckResults(w io.Writer, results []checker.Result, countries []string, format Format) error {
	rows := make([]checkRow, len(results))
	for i, r := range results {
		c := ""
		if i < len(countries) {
			c = countries[i]
		}
		rows[i] = toCheckRow(r, c)
	}

	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	case FormatCSV:
		cw := csv.NewWriter(w)
		cw.Write([]string{"address", "protocol", "alive", "latency_ms", "country", "error"}) //nolint:errcheck
		for _, row := range rows {
			cw.Write([]string{
				row.Address,
				row.Protocol,
				strconv.FormatBool(row.Alive),
				strconv.FormatInt(row.LatencyMS, 10),
				row.Country,
				row.Error,
			}) //nolint:errcheck
		}
		cw.Flush()
		return cw.Error()
	default: // table
		fmt.Fprintf(w, "%-45s %-8s %-6s %8s  %-15s  %s\n",
			"ADDRESS", "PROTO", "ALIVE", "LAT(ms)", "COUNTRY", "ERROR")
		fmt.Fprintf(w, "%s\n", repeat('-', 110))
		for _, row := range rows {
			alive := "✗"
			if row.Alive {
				alive = "✓"
			}
			fmt.Fprintf(w, "%-45s %-8s %-6s %8d  %-15s  %s\n",
				truncate(row.Address, 45),
				row.Protocol,
				alive,
				row.LatencyMS,
				row.Country,
				row.Error,
			)
		}
		return nil
	}
}

// ---- Bench results ----------------------------------------------------------

// WriteBenchResults writes benchmark stats in the requested format.
func WriteBenchResults(w io.Writer, results []bench.Stats, format Format) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	case FormatCSV:
		cw := csv.NewWriter(w)
		cw.Write([]string{"address", "samples", "successful", "min_ms", "max_ms", "avg_ms", "p50_ms", "p95_ms", "loss_rate", "speed_bps"}) //nolint:errcheck
		for _, r := range results {
			cw.Write([]string{
				r.Address,
				strconv.Itoa(r.Samples),
				strconv.Itoa(r.Successful),
				strconv.FormatInt(r.MinMS, 10),
				strconv.FormatInt(r.MaxMS, 10),
				strconv.FormatInt(r.AvgMS, 10),
				strconv.FormatInt(r.P50MS, 10),
				strconv.FormatInt(r.P95MS, 10),
				strconv.FormatFloat(r.LossRate, 'f', 4, 64),
				strconv.FormatInt(r.SpeedBps, 10),
			}) //nolint:errcheck
		}
		cw.Flush()
		return cw.Error()
	default: // table
		fmt.Fprintf(w, "%-45s %4s %4s %7s %7s %7s %7s %7s %8s\n",
			"ADDRESS", "OK", "ERR", "MIN", "AVG", "P50", "P95", "MAX", "LOSS%")
		fmt.Fprintf(w, "%s\n", repeat('-', 115))
		for _, r := range results {
			failed := r.Samples - r.Successful
			fmt.Fprintf(w, "%-45s %4d %4d %7d %7d %7d %7d %7d %7.1f%%\n",
				truncate(r.Address, 45),
				r.Successful, failed,
				r.MinMS, r.AvgMS, r.P50MS, r.P95MS, r.MaxMS,
				r.LossRate*100,
			)
		}
		return nil
	}
}

// helpers

func repeat(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
