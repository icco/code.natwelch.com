package code

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

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
