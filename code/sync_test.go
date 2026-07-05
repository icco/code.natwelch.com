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
