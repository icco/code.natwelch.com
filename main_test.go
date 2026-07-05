package main

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestEnvOr(t *testing.T) {
	const key = "CODE_TEST_ENVOR"
	os.Unsetenv(key)
	if got := envOr(key, "fallback"); got != "fallback" {
		t.Errorf("envOr(unset) = %q, want fallback", got)
	}
	os.Setenv(key, "value")
	defer os.Unsetenv(key)
	if got := envOr(key, "fallback"); got != "value" {
		t.Errorf("envOr(set) = %q, want value", got)
	}
}

func TestWriteCSV(t *testing.T) {
	rec := httptest.NewRecorder()
	writeCSV(rec, "date", map[string]int64{"2024-03-02": 1, "2024-03-01": 5})

	want := "date,count\n2024-03-01,5\n2024-03-02,1\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("writeCSV body = %q, want %q", got, want)
	}
	if ct := rec.Header().Get("content-type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("content-type = %q, want text/csv", ct)
	}
}
