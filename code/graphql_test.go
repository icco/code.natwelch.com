package code

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap"
)

func TestFetchCommitContributionsNon200IncludesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Bad credentials"))
	}))
	defer srv.Close()

	old := graphqlURL
	graphqlURL = srv.URL
	defer func() { graphqlURL = old }()

	_, err := FetchCommitContributions(context.Background(), zap.NewNop().Sugar(), "tkn", "icco", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected an error on HTTP 401")
	}
	if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "Bad credentials") {
		t.Errorf("error should carry status and body, got: %v", err)
	}
}

func TestFetchCommitContributionsGraphQLErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"Could not resolve to a User"}]}`))
	}))
	defer srv.Close()

	old := graphqlURL
	graphqlURL = srv.URL
	defer func() { graphqlURL = old }()

	_, err := FetchCommitContributions(context.Background(), zap.NewNop().Sugar(), "tkn", "nobody", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected an error when the response carries a GraphQL errors array")
	}
	if !strings.Contains(err.Error(), "Could not resolve to a User") {
		t.Errorf("error should include the GraphQL message, got: %v", err)
	}
}

func TestFetchCommitContributionsAggregates(t *testing.T) {
	const payload = `{"data":{"user":{"contributionsCollection":{
	  "commitContributionsByRepository":[
	    {"contributions":{"nodes":[
	      {"occurredAt":"2024-03-01T00:00:00Z","commitCount":2},
	      {"occurredAt":"2024-03-02T00:00:00Z","commitCount":1}],
	      "pageInfo":{"hasNextPage":false}}},
	    {"contributions":{"nodes":[
	      {"occurredAt":"2024-03-01T00:00:00Z","commitCount":3}],
	      "pageInfo":{"hasNextPage":false}}}
	  ]}}}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "bearer tkn" {
			t.Errorf("auth header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	old := graphqlURL
	graphqlURL = srv.URL
	defer func() { graphqlURL = old }()

	from := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	got, err := FetchCommitContributions(context.Background(), zap.NewNop().Sugar(), "tkn", "icco", from, from.AddDate(0, 1, 0))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got["2024-03-01"] != 5 || got["2024-03-02"] != 1 {
		t.Errorf("aggregation wrong: %v", got)
	}
}

// fetchPayload serves a fixed GraphQL body and runs one FetchCommitContributions.
func fetchPayload(t *testing.T, payload string) map[string]int {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	old := graphqlURL
	graphqlURL = srv.URL
	defer func() { graphqlURL = old }()

	from := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	got, err := FetchCommitContributions(context.Background(), zap.NewNop().Sugar(), "tkn", "icco", from, from.AddDate(0, 1, 0))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	return got
}

func TestFetchCommitContributionsPrivateVisibility(t *testing.T) {
	// A private repo in the itemized list is counted and marks the token healthy.
	got := fetchPayload(t, `{"data":{"user":{"contributionsCollection":{
	  "restrictedContributionsCount":0,
	  "commitContributionsByRepository":[
	    {"repository":{"isPrivate":true},"contributions":{"nodes":[
	      {"occurredAt":"2024-03-01T00:00:00Z","commitCount":4}],"pageInfo":{"hasNextPage":false}}}
	  ]}}}}`)
	if got["2024-03-01"] != 4 {
		t.Errorf("private commit not aggregated: %v", got)
	}
	if v := testutil.ToFloat64(MetricPrivateVisible); v != 1 {
		t.Errorf("private commits itemized: want visible=1, got %v", v)
	}

	// Restricted activity with no itemized private commits => dropped => 0.
	fetchPayload(t, `{"data":{"user":{"contributionsCollection":{
	  "restrictedContributionsCount":7,
	  "commitContributionsByRepository":[
	    {"repository":{"isPrivate":false},"contributions":{"nodes":[
	      {"occurredAt":"2024-03-01T00:00:00Z","commitCount":2}],"pageInfo":{"hasNextPage":false}}}
	  ]}}}}`)
	if v := testutil.ToFloat64(MetricPrivateVisible); v != 0 {
		t.Errorf("restricted-but-none-itemized: want visible=0, got %v", v)
	}
}
