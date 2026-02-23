package geo

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Source defines a free, no-auth IP-to-country CSV database that can be
// downloaded automatically. The URL may contain a {YYYY-MM} placeholder
// that is replaced with the current year-month.
type Source struct {
	Name    string
	URL     string
	Gzipped bool
}

// BuiltinSources lists the default free IP-country databases.
var BuiltinSources = []Source{
	{
		Name:    "db-ip-country-lite",
		URL:     "https://download.db-ip.com/free/dbip-country-lite-{YYYY-MM}.csv.gz",
		Gzipped: true,
	},
}

// UpdateOptions configures a database update run.
type UpdateOptions struct {
	Source   *Source       // nil = first BuiltinSources entry
	DestPath string        // path to write; "" = DefaultDBPath()
	Timeout  time.Duration // HTTP timeout; 0 = 60s
	Progress func(msg string)
}

// Update downloads a fresh IP-country CSV and writes it to DestPath.
// It replaces the existing file atomically (write to temp, then rename).
func Update(opts UpdateOptions) error {
	if opts.Timeout == 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.DestPath == "" {
		opts.DestPath = DefaultDBPath()
	}
	src := opts.Source
	if src == nil {
		s := BuiltinSources[0]
		src = &s
	}

	// Expand {YYYY-MM} placeholder.
	now := time.Now().UTC()
	rawURL := expandURL(src.URL, now)

	log := opts.Progress
	if log == nil {
		log = func(string) {}
	}

	log(fmt.Sprintf("Downloading %s from %s …", src.Name, rawURL))

	client := &http.Client{Timeout: opts.Timeout}
	resp, err := client.Get(rawURL)
	if err != nil {
		// If current month fails, try previous month (db-ip publishes on ~1st).
		prev := now.AddDate(0, -1, 0)
		rawURL = expandURL(src.URL, prev)
		log(fmt.Sprintf("Retrying with previous month: %s …", rawURL))
		resp, err = client.Get(rawURL)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}

	var reader io.Reader = resp.Body
	if src.Gzipped {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("gzip: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	// Ensure destination directory exists.
	if err := os.MkdirAll(filepath.Dir(opts.DestPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Write to a temp file then atomically rename.
	tmp := opts.DestPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}

	n, err := io.Copy(f, reader)
	f.Close()
	if err != nil {
		os.Remove(tmp) //nolint:errcheck
		return fmt.Errorf("write: %w", err)
	}

	if err := os.Rename(tmp, opts.DestPath); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return fmt.Errorf("rename: %w", err)
	}

	log(fmt.Sprintf("Saved %.1f MB → %s", float64(n)/(1<<20), opts.DestPath))
	return nil
}

// expandURL replaces {YYYY-MM} with the formatted month string.
func expandURL(tmpl string, t time.Time) string {
	return replaceAll(tmpl, "{YYYY-MM}", t.Format("2006-01"))
}

// replaceAll is a simple string replacement (avoids importing strings just for this).
func replaceAll(s, old, new string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if len(s[i:]) >= len(old) && s[i:i+len(old)] == old {
			out = append(out, new...)
			i += len(old)
		} else {
			out = append(out, s[i])
			i++
		}
	}
	return string(out)
}
