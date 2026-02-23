// Package geo provides IP-to-country lookups using a local CSV database.
// The database is loaded lazily from the default data path on first use.
// Use DB.Load() / DB.LoadFile() explicitly if you need early error handling.
package geo

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Entry represents a single IP range → country mapping.
type Entry struct {
	Start       uint32
	End         uint32
	CountryCode string
	CountryName string
}

// DB is a loaded geo database.
type DB struct {
	mu      sync.RWMutex
	entries []Entry
	loaded  bool
}

// DefaultDB is the package-level singleton, loaded lazily.
var DefaultDB = &DB{}

// DefaultDBPath returns the canonical path to the bundled IP database file.
func DefaultDBPath() string {
	// Walk up from this file's package dir to find the project root's data/ folder.
	_, file, _, ok := runtime.Caller(0)
	if ok {
		root := filepath.Join(filepath.Dir(file), "..", "..", "data")
		p := filepath.Join(root, "ip2country.csv")
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
	}
	// Fallback: same dir as the binary.
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "data", "ip2country.csv")
}

// Load loads the database from the default path.
func (db *DB) Load() error {
	return db.LoadFile(DefaultDBPath())
}

// LoadFile parses a CSV file in the format:
//
//	ip_from,ip_to,country_code,country_name
//
// Lines starting with # are ignored.
func (db *DB) LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip optional quotes.
		line = strings.ReplaceAll(line, "\"", "")
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue // skip malformed
		}
		start, err := parseIP(parts[0])
		if err != nil {
			continue
		}
		end, err := parseIP(parts[1])
		if err != nil {
			continue
		}
		cc := strings.TrimSpace(parts[2])
		cn := ""
		if len(parts) >= 4 {
			cn = strings.TrimSpace(parts[3])
		}
		entries = append(entries, Entry{
			Start:       start,
			End:         end,
			CountryCode: cc,
			CountryName: cn,
		})
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	// Sort by start IP for binary search.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Start < entries[j].Start
	})

	db.mu.Lock()
	db.entries = entries
	db.loaded = true
	db.mu.Unlock()
	return nil
}

// Lookup returns the country for an IP string. Returns ("--","Unknown") if not found.
func (db *DB) Lookup(ipStr string) (countryCode, countryName string) {
	db.mu.RLock()
	if !db.loaded {
		db.mu.RUnlock()
		db.Load() //nolint:errcheck — best effort
		db.mu.RLock()
	}
	defer db.mu.RUnlock()

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "--", "Unknown"
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return "--", "IPv6 not supported"
	}
	n := binary.BigEndian.Uint32(ip4)

	idx := sort.Search(len(db.entries), func(i int) bool {
		return db.entries[i].End >= n
	})
	if idx < len(db.entries) && db.entries[idx].Start <= n && n <= db.entries[idx].End {
		return db.entries[idx].CountryCode, db.entries[idx].CountryName
	}
	return "--", "Unknown"
}

// Loaded returns true if the database has been loaded.
func (db *DB) Loaded() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.loaded
}

// Count returns the number of entries in the database.
func (db *DB) Count() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.entries)
}

// Lookup is a convenience wrapper around DefaultDB.Lookup.
func Lookup(ipStr string) (code, name string) {
	return DefaultDB.Lookup(ipStr)
}

// parseIP handles both dotted-decimal IPv4 strings ("1.2.3.4")
// and numeric integer strings ("16909060").
func parseIP(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	// Numeric?
	if n, err := strconv.ParseUint(s, 10, 32); err == nil {
		return uint32(n), nil
	}
	// Dotted decimal.
	ip := net.ParseIP(s)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP: %s", s)
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return 0, fmt.Errorf("not IPv4: %s", s)
	}
	return binary.BigEndian.Uint32(ip4), nil
}
