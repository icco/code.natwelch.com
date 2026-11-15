package code

import (
	"testing"
	"time"
)

func TestMissingYears(t *testing.T) {
	got := missingYears(map[int]bool{2009: true, 2011: true}, 2008, 2012)
	want := []int{2008, 2010, 2012}
	if len(got) != len(want) {
		t.Fatalf("missingYears = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("missingYears = %v, want %v", got, want)
		}
	}

	if n := len(missingYears(map[int]bool{2020: true, 2021: true}, 2020, 2021)); n != 0 {
		t.Errorf("all-present should yield 0 missing, got %d", n)
	}
	if n := len(missingYears(nil, 2008, 2010)); n != 3 {
		t.Errorf("nil have should yield every year, got %d", n)
	}
}

func TestRefreshStart(t *testing.T) {
	now := time.Date(2024, 3, 15, 13, 45, 0, 0, time.UTC)
	want := time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)
	if got := refreshStart(now); !got.Equal(want) {
		t.Errorf("refreshStart(%v) = %v, want %v", now, got, want)
	}
}

func TestWindowRowsZeroFills(t *testing.T) {
	from := time.Date(2024, 2, 27, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 1, 9, 0, 0, 0, time.UTC)
	rows := windowRows("alice", from, to, map[string]int{"2024-02-28": 3})

	got := map[string]int{}
	for _, r := range rows {
		if r.User != "alice" {
			t.Errorf("row user = %q, want alice", r.User)
		}
		got[r.Date.Format("2006-01-02")] = r.Count
	}
	want := map[string]int{"2024-02-27": 0, "2024-02-28": 3, "2024-02-29": 0, "2024-03-01": 0}
	if len(got) != len(want) {
		t.Fatalf("windowRows = %v, want %v", got, want)
	}
	for d, c := range want {
		if n, ok := got[d]; !ok || n != c {
			t.Errorf("windowRows[%s] = %d (present %v), want %d", d, n, ok, c)
		}
	}
}

func TestMonthlyWindows(t *testing.T) {
	from := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	got := monthlyWindows(from, to)

	if len(got) != 3 {
		t.Fatalf("got %d windows, want 3: %v", len(got), got)
	}
	if !got[0][0].Equal(from) {
		t.Errorf("first window start = %v, want %v", got[0][0], from)
	}
	if !got[len(got)-1][1].Equal(to) {
		t.Errorf("last window end = %v, want %v", got[len(got)-1][1], to)
	}
	for i, w := range got {
		if !w[1].After(w[0]) {
			t.Errorf("window %d not forward: %v", i, w)
		}
	}
}
