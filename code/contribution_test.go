package code

import (
	"testing"
	"time"
)

func TestWeekKey(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"2024-01-01", "2024-W01"}, // Monday, ISO week 1
		{"2021-01-01", "2020-W53"}, // Friday belongs to prior ISO year
		{"2024-12-31", "2025-W01"}, // Tuesday rolls into next ISO year
	}
	for _, c := range cases {
		d, err := time.Parse("2006-01-02", c.in)
		if err != nil {
			t.Fatalf("parse %s: %v", c.in, err)
		}
		if got := weekKey(d); got != c.want {
			t.Errorf("weekKey(%s) = %q, want %q", c.in, got, c.want)
		}
	}
}
