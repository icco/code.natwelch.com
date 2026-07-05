package code

import (
	"testing"
	"time"
)

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
