package bench

import (
	"testing"
)

func TestAvg(t *testing.T) {
	cases := []struct {
		vals []int64
		want int64
	}{
		{[]int64{10, 20, 30}, 20},
		{[]int64{100}, 100},
		{[]int64{1, 2, 3, 4, 5}, 3},
	}
	for _, c := range cases {
		got := avg(c.vals)
		if got != c.want {
			t.Errorf("avg(%v) = %d, want %d", c.vals, got, c.want)
		}
	}
}

func TestPercentile(t *testing.T) {
	sorted := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	cases := []struct {
		p    int
		want int64
	}{
		{50, 60},
		{95, 100},
		{0, 10},
	}
	for _, c := range cases {
		got := percentile(sorted, c.p)
		if got != c.want {
			t.Errorf("percentile(%v, %d) = %d, want %d", sorted, c.p, got, c.want)
		}
	}
}

func TestPercentile_empty(t *testing.T) {
	got := percentile(nil, 50)
	if got != 0 {
		t.Errorf("percentile(nil,50) = %d, want 0", got)
	}
}

func TestRunMany_emptyInput(t *testing.T) {
	results := RunMany(nil, DefaultOptions())
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.Samples != 5 {
		t.Errorf("default samples = %d, want 5", opts.Samples)
	}
}

func TestRun_zeroSamples(t *testing.T) {
	// Run with samples=0 should coerce to 5 and not panic.
	opts := DefaultOptions()
	opts.Samples = 0
	// Use an invalid address — we just want no panic.
	stats := Run("http://127.0.0.1:1", opts)
	if stats.Address != "http://127.0.0.1:1" {
		t.Errorf("address not preserved")
	}
}
