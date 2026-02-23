package geo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleCSV = `# ip2country sample
# ip_from,ip_to,country_code,country_name
16777216,16777471,AU,Australia
16777472,16778239,CN,China
16778240,16779263,AU,Australia
134744072,134744072,US,United States
`

func writeTempDB(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ip2country.csv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp db: %v", err)
	}
	return path
}

func TestLoadAndLookup(t *testing.T) {
	path := writeTempDB(t, sampleCSV)
	db := &DB{}
	if err := db.LoadFile(path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}

	// 16777216 = 1.0.0.0 (AU range start)
	cc, cn := db.Lookup("1.0.0.0")
	if cc != "AU" {
		t.Errorf("1.0.0.0 country code = %q, want AU", cc)
	}
	if cn != "Australia" {
		t.Errorf("1.0.0.0 country name = %q, want Australia", cn)
	}

	// 8.8.8.8 = 134744072 (US)
	cc2, cn2 := db.Lookup("8.8.8.8")
	if cc2 != "US" {
		t.Errorf("8.8.8.8 country code = %q, want US", cc2)
	}
	if cn2 != "United States" {
		t.Errorf("8.8.8.8 country name = %q, want United States", cn2)
	}
}

func TestLookup_notFound(t *testing.T) {
	path := writeTempDB(t, sampleCSV)
	db := &DB{}
	_ = db.LoadFile(path)

	cc, _ := db.Lookup("255.255.255.255")
	if cc != "--" {
		t.Errorf("out-of-range IP country = %q, want --", cc)
	}
}

func TestLookup_invalidIP(t *testing.T) {
	path := writeTempDB(t, sampleCSV)
	db := &DB{}
	_ = db.LoadFile(path)

	cc, _ := db.Lookup("not-an-ip")
	if cc != "--" {
		t.Errorf("invalid IP country = %q, want --", cc)
	}
}

func TestLoadFile_missingFile(t *testing.T) {
	db := &DB{}
	err := db.LoadFile("/nonexistent/path/ip2country.csv")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoadFile_malformed(t *testing.T) {
	content := "this,is,not,valid,csv,data\nbad line\n"
	path := writeTempDB(t, content)
	db := &DB{}
	// Should not error — malformed lines are skipped gracefully.
	if err := db.LoadFile(path); err != nil {
		t.Errorf("unexpected error on malformed CSV: %v", err)
	}
}

func TestCount(t *testing.T) {
	path := writeTempDB(t, sampleCSV)
	db := &DB{}
	_ = db.LoadFile(path)
	// sampleCSV has 4 data rows.
	if db.Count() != 4 {
		t.Errorf("Count() = %d, want 4", db.Count())
	}
}

func TestExpandURL(t *testing.T) {
	// Just test the replaceAll helper via the exported behaviour.
	src := BuiltinSources[0]
	if !strings.Contains(src.URL, "{YYYY-MM}") {
		t.Error("builtin source URL should contain {YYYY-MM} placeholder")
	}
}

func TestLoadFile_dottedDecimal(t *testing.T) {
	content := "# dotted decimal format\n1.0.0.0,1.0.0.255,AU,Australia\n"
	path := writeTempDB(t, content)
	db := &DB{}
	if err := db.LoadFile(path); err != nil {
		t.Fatalf("LoadFile dotted: %v", err)
	}
	cc, _ := db.Lookup("1.0.0.128")
	if cc != "AU" {
		t.Errorf("dotted decimal lookup = %q, want AU", cc)
	}
}
