package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/romeomihailus/proxybench/internal/bench"
	"github.com/romeomihailus/proxybench/internal/checker"
)

func makeCheckResults() []checker.Result {
	return []checker.Result{
		{
			Address:  "http://1.2.3.4:8080",
			Protocol: checker.ProtocolHTTP,
			Alive:    true,
			Latency:  200 * time.Millisecond,
		},
		{
			Address:  "socks5://5.6.7.8:1080",
			Protocol: checker.ProtocolSOCKS5,
			Alive:    false,
			Error:    "connection refused",
		},
	}
}

func makeBenchResults() []bench.Stats {
	return []bench.Stats{
		{
			Address:    "http://1.2.3.4:8080",
			Samples:    5,
			Successful: 4,
			MinMS:      100,
			MaxMS:      400,
			AvgMS:      200,
			P50MS:      190,
			P95MS:      380,
			LossRate:   0.2,
		},
	}
}

// ---- Check: JSON ------------------------------------------------------------

func TestWriteCheckResults_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCheckResults(&buf, makeCheckResults(), []string{"US United States", ""}, FormatJSON)
	if err != nil {
		t.Fatalf("WriteCheckResults JSON: %v", err)
	}
	var rows []checkRow
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Alive != true {
		t.Error("first row should be alive")
	}
	if rows[0].LatencyMS != 200 {
		t.Errorf("latency = %d, want 200", rows[0].LatencyMS)
	}
	if rows[0].Country != "US United States" {
		t.Errorf("country = %q, want US United States", rows[0].Country)
	}
	if rows[1].Error != "connection refused" {
		t.Errorf("error field = %q, want 'connection refused'", rows[1].Error)
	}
}

// ---- Check: CSV -------------------------------------------------------------

func TestWriteCheckResults_CSV(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCheckResults(&buf, makeCheckResults(), nil, FormatCSV)
	if err != nil {
		t.Fatalf("WriteCheckResults CSV: %v", err)
	}
	r := csv.NewReader(strings.NewReader(buf.String()))
	records, _ := r.ReadAll()
	if len(records) != 3 { // header + 2 rows
		t.Errorf("expected 3 records, got %d", len(records))
	}
	if records[0][0] != "address" {
		t.Errorf("first column header = %q, want address", records[0][0])
	}
}

// ---- Check: Table -----------------------------------------------------------

func TestWriteCheckResults_Table(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCheckResults(&buf, makeCheckResults(), nil, FormatTable)
	if err != nil {
		t.Fatalf("WriteCheckResults Table: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "ADDRESS") {
		t.Error("table output should contain ADDRESS header")
	}
	if !strings.Contains(out, "✓") {
		t.Error("table output should contain ✓ for alive proxy")
	}
	if !strings.Contains(out, "✗") {
		t.Error("table output should contain ✗ for dead proxy")
	}
}

// ---- Bench: JSON ------------------------------------------------------------

func TestWriteBenchResults_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := WriteBenchResults(&buf, makeBenchResults(), FormatJSON)
	if err != nil {
		t.Fatalf("WriteBenchResults JSON: %v", err)
	}
	var rows []bench.Stats
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rows[0].P95MS != 380 {
		t.Errorf("p95 = %d, want 380", rows[0].P95MS)
	}
}

// ---- Bench: CSV -------------------------------------------------------------

func TestWriteBenchResults_CSV(t *testing.T) {
	var buf bytes.Buffer
	err := WriteBenchResults(&buf, makeBenchResults(), FormatCSV)
	if err != nil {
		t.Fatalf("WriteBenchResults CSV: %v", err)
	}
	r := csv.NewReader(strings.NewReader(buf.String()))
	records, _ := r.ReadAll()
	if len(records) != 2 { // header + 1 row
		t.Errorf("expected 2 records, got %d", len(records))
	}
	if records[1][0] != "http://1.2.3.4:8080" {
		t.Errorf("address field = %q", records[1][0])
	}
}

// ---- Bench: Table -----------------------------------------------------------

func TestWriteBenchResults_Table(t *testing.T) {
	var buf bytes.Buffer
	err := WriteBenchResults(&buf, makeBenchResults(), FormatTable)
	if err != nil {
		t.Fatalf("WriteBenchResults Table: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "AVG") {
		t.Error("bench table should contain AVG column")
	}
}

// ---- helpers ----------------------------------------------------------------

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate short = %q", got)
	}
	long := "abcdefghijklmnopqrstuvwxyz"
	if got := truncate(long, 5); len([]rune(got)) > 5 {
		t.Errorf("truncate long too long: %q", got)
	}
}
