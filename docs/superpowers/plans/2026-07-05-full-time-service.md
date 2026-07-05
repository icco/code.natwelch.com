# code.natwelch.com Full-Time Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `code.natwelch.com` own its own commit-contribution data pipeline (GraphQL, in-process), self-render its chart, and publish `ghcr.io/icco/code.natwelch.com:main` from CI.

**Architecture:** One Go binary serves HTTP on `:8080` and runs a background sync goroutine. The goroutine pulls commits-only daily counts from GitHub's GraphQL `contributionsCollection.commitContributionsByRepository`, backfills all history on boot, then refreshes a trailing window on an interval, upserting into a `contributions` table (self-migrated on boot). The static site renders an inline-SVG calendar heatmap from a CSV endpoint — no external notebook or CDN.

**Tech Stack:** Go 1.26, chi/v5, GORM + Postgres, `prometheus/client_golang`, raw `net/http` GraphQL client, vanilla JS/SVG, Docker (multi-stage), GitHub Actions, Taskfile.

## Global Constraints

- Go toolchain **1.26**; `go.mod` `go 1.26.0`.
- Image is `ghcr.io/icco/code.natwelch.com:main`; binary listens on `:8080`.
- **Do NOT edit `icco.me`** — match its already-wired deploy contract only.
- Data source is GitHub **GraphQL, commits only** (`commitContributionsByRepository`); never the all-types calendar. No REST commit crawl.
- **Retire** `POST /save`, `code.Commit`, `CheckAndSave`, `UserRepos`, the old `go-github/v37` + `oauth2` deps.
- Viz is **self-hosted**: no `observablehq.com`, no CDN scripts.
- Build automation is a **Taskfile.yml**, not a Makefile.
- Host-side steps (rope DB, real PAT) are a **runbook only** — never commit real secrets; never edit `icco.me`.
- Sync config from env: `GITHUB_USER` (default `icco`), `GITHUB_TOKEN`, `SYNC_INTERVAL` (default `6h`), `SYNC_START_YEAR` (default `2008`).

## File Structure

| File | Responsibility | Task |
|------|----------------|------|
| `code/contribution.go` | `Contribution` model, `Upsert`, `ForAllTime`, `ForYear`, `weekKey` | 1 |
| `code/contribution_test.go` | `weekKey` unit tests | 1 |
| `code/graphql.go` | `FetchCommitContributions` GraphQL client (raw HTTP) | 2 |
| `code/graphql_test.go` | client aggregation test (httptest) | 2 |
| `code/metrics.go` | prometheus collectors | 3 |
| `code/sync.go` | `RunSync`, `syncRange`, `monthlyWindows`, `SyncOptions` | 3 |
| `code/sync_test.go` | `monthlyWindows` unit test | 3 |
| `main.go` | server wiring, endpoints, `/metrics`, graceful shutdown, launch sync | 4 |
| `code/commit.go` | **DELETE** | 4 |
| `code/github.go` | **DELETE** (old REST client) | 4 |
| `static/index.html` | page shell (no Observable) | 5 |
| `static/app.js` | fetch CSV → inline-SVG calendar heatmap | 5 |
| `Dockerfile` | multi-stage Go 1.26 build | 6 |
| `Taskfile.yml` | build/test/lint/run | 6 |
| `.travis.yml` | **DELETE** | 6 |
| `.github/workflows/docker.yml` | build+push ghcr `:main` | 7 |
| `.github/workflows/test.yml` | test + vet | 7 |
| `README.md` | rewrite | 8 |
| `docs/deploy.md` | operator runbook | 8 |

---

### Task 1: `Contribution` model + queries

**Files:**
- Create: `code/contribution.go`
- Test: `code/contribution_test.go`

**Interfaces:**
- Produces: `code.Contribution` struct; `code.Upsert(ctx, db, []Contribution) error`; `code.ForAllTime(ctx, db, user) (map[string]int64, error)` (day `2006-01-02` → count); `code.ForYear(ctx, db, user, year) (map[string]int64, error)` (ISO-week key → summed count); `code.weekKey(time.Time) string`.
- Consumes: nothing (additive; `commit.go` stays until Task 4).

- [ ] **Step 1: Write the failing test**

`code/contribution_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd ~/Projects/code.natwelch.com && go test ./code/ -run TestWeekKey -v`
Expected: FAIL — `undefined: weekKey`.

- [ ] **Step 3: Write the implementation**

`code/contribution.go`:
```go
package code

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Contribution is a per-day commit count for a GitHub user.
type Contribution struct {
	ID        uint      `gorm:"primarykey" json:"-"`
	User      string    `gorm:"index:idx_user_date,unique" json:"user"`
	Date      time.Time `gorm:"index:idx_user_date,unique;type:date" json:"date"`
	Count     int       `json:"count"`
	UpdatedAt time.Time `json:"-"`
}

// Upsert inserts or updates daily counts, keyed on (user, date).
func Upsert(ctx context.Context, db *gorm.DB, rows []Contribution) error {
	if len(rows) == 0 {
		return nil
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"count", "updated_at"}),
	}).Create(&rows).Error
}

// ForAllTime returns every stored day (as "2006-01-02") mapped to its count.
func ForAllTime(ctx context.Context, db *gorm.DB, user string) (map[string]int64, error) {
	var rows []Contribution
	if err := db.WithContext(ctx).Where(`"user" = ?`, user).Order("date asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Date.Format("2006-01-02")] = int64(r.Count)
	}
	return out, nil
}

// ForYear returns counts for one calendar year, grouped by ISO week key.
func ForYear(ctx context.Context, db *gorm.DB, user string, year int) (map[string]int64, error) {
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(1, 0, 0)
	var rows []Contribution
	if err := db.WithContext(ctx).
		Where(`"user" = ? AND date >= ? AND date < ?`, user, from, to).
		Order("date asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[weekKey(r.Date)] += int64(r.Count)
	}
	return out, nil
}

func weekKey(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", y, w)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./code/ -run TestWeekKey -v`
Expected: PASS. Then `go build ./...` → succeeds (commit.go/github.go untouched).

- [ ] **Step 5: Commit**

```bash
git add code/contribution.go code/contribution_test.go
git commit -m "feat: add Contribution model and daily-count queries"
```

---

### Task 2: GraphQL commit-contributions client

**Files:**
- Create: `code/graphql.go`
- Test: `code/graphql_test.go`

**Interfaces:**
- Produces: `code.FetchCommitContributions(ctx, log *zap.SugaredLogger, token, user string, from, to time.Time) (map[string]int, error)` — one GraphQL window (`from`/`to` ≤ ~3 months); returns day `2006-01-02` → summed commit count across repos. Also `code.graphqlURL` const and internal `contribResponse` type.
- Consumes: nothing (additive; leaves old `github.go` in place until Task 4).

- [ ] **Step 1: Write the failing test**

`code/graphql_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./code/ -run TestFetchCommitContributions -v`
Expected: FAIL — `undefined: graphqlURL` / `FetchCommitContributions`.

- [ ] **Step 3: Write the implementation**

`code/graphql.go`:
```go
package code

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// graphqlURL is a var so tests can point it at httptest.
var graphqlURL = "https://api.github.com/graphql"

const contribQuery = `query($login:String!,$from:DateTime!,$to:DateTime!){
  user(login:$login){
    contributionsCollection(from:$from,to:$to){
      commitContributionsByRepository(maxRepositories:100){
        contributions(first:100){
          nodes{ occurredAt commitCount }
          pageInfo{ hasNextPage }
        }
      }
    }
  }
}`

type contribResponse struct {
	Data struct {
		User struct {
			ContributionsCollection struct {
				CommitContributionsByRepository []struct {
					Contributions struct {
						Nodes []struct {
							OccurredAt  time.Time `json:"occurredAt"`
							CommitCount int       `json:"commitCount"`
						} `json:"nodes"`
						PageInfo struct {
							HasNextPage bool `json:"hasNextPage"`
						} `json:"pageInfo"`
					} `json:"contributions"`
				} `json:"commitContributionsByRepository"`
			} `json:"contributionsCollection"`
		} `json:"user"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// FetchCommitContributions returns day("2006-01-02") -> commit count for a
// single window (keep windows <= ~3 months so first:100 never truncates).
func FetchCommitContributions(ctx context.Context, log *zap.SugaredLogger, token, user string, from, to time.Time) (map[string]int, error) {
	reqBody, err := json.Marshal(map[string]any{
		"query": contribQuery,
		"variables": map[string]any{
			"login": user,
			"from":  from.UTC().Format(time.RFC3339),
			"to":    to.UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql status %d", resp.StatusCode)
	}

	var parsed contribResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", parsed.Errors[0].Message)
	}

	repos := parsed.Data.User.ContributionsCollection.CommitContributionsByRepository
	if len(repos) >= 100 {
		log.Warnw("commitContributionsByRepository hit maxRepositories cap; some repos may be omitted", "from", from, "to", to)
	}

	out := map[string]int{}
	for _, repo := range repos {
		if repo.Contributions.PageInfo.HasNextPage {
			log.Warnw("commit contributions truncated for window; narrow it", "from", from, "to", to)
		}
		for _, n := range repo.Contributions.Nodes {
			out[n.OccurredAt.UTC().Format("2006-01-02")] += n.CommitCount
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./code/ -run TestFetchCommitContributions -v`
Expected: PASS. `go build ./...` → succeeds.

- [ ] **Step 5: Commit**

```bash
git add code/graphql.go code/graphql_test.go
git commit -m "feat: add GraphQL commit-contributions client"
```

---

### Task 3: prometheus metrics + sync scheduler

**Files:**
- Create: `code/metrics.go`, `code/sync.go`
- Test: `code/sync_test.go`

**Interfaces:**
- Produces: `code.SyncOptions{User, Token string; Interval time.Duration; StartYear int}`; `code.RunSync(ctx, log, db, opts)` (blocking until ctx done — call as goroutine); `code.monthlyWindows(from, to) [][2]time.Time`; metrics `MetricLastSync`, `MetricSyncErrors`, `MetricContributions`.
- Consumes: `Contribution`, `Upsert` (Task 1); `FetchCommitContributions` (Task 2).

- [ ] **Step 1: Add the prometheus dependency**

Run: `go get github.com/prometheus/client_golang@v1.23.2`
Expected: `go.mod` gains `github.com/prometheus/client_golang`.

- [ ] **Step 2: Write the failing test**

`code/sync_test.go`:
```go
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
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./code/ -run TestMonthlyWindows -v`
Expected: FAIL — `undefined: monthlyWindows`.

- [ ] **Step 4: Write the metrics**

`code/metrics.go`:
```go
package code

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MetricLastSync is the unix time of the last completed sync window.
	MetricLastSync = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "code_last_sync_timestamp_seconds",
		Help: "Unix timestamp of the last completed sync window.",
	})
	// MetricSyncErrors counts window fetch/upsert failures.
	MetricSyncErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "code_sync_errors_total",
		Help: "Total number of sync window errors.",
	})
	// MetricContributions is the commit total from the most recent backfill.
	MetricContributions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "code_contributions_total",
		Help: "Commit total observed during the most recent full backfill.",
	})
)
```

- [ ] **Step 5: Write the sync scheduler**

`code/sync.go`:
```go
package code

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SyncOptions configures the background sync loop.
type SyncOptions struct {
	User      string
	Token     string
	Interval  time.Duration
	StartYear int
}

// RunSync backfills all history once, then refreshes a trailing window on
// Interval until ctx is cancelled. Intended to run as a goroutine.
func RunSync(ctx context.Context, log *zap.SugaredLogger, db *gorm.DB, opts SyncOptions) {
	if opts.Token == "" {
		log.Warnw("GITHUB_TOKEN is empty; contribution sync disabled")
		return
	}

	log.Infow("starting contribution sync", "user", opts.User, "interval", opts.Interval, "start_year", opts.StartYear)
	backfillFrom := time.Date(opts.StartYear, 1, 1, 0, 0, 0, 0, time.UTC)
	if total := syncRange(ctx, log, db, opts, backfillFrom, time.Now().UTC()); total >= 0 {
		MetricContributions.Set(float64(total))
	}

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Infow("contribution sync shutting down")
			return
		case <-ticker.C:
			// Trailing 13 months catches backdated / late commits.
			from := time.Now().UTC().AddDate(-1, -1, 0)
			syncRange(ctx, log, db, opts, from, time.Now().UTC())
		}
	}
}

// syncRange fetches each monthly window in [from,to) and upserts it. Returns
// the summed commit count, or -1 if the range was aborted before any window.
func syncRange(ctx context.Context, log *zap.SugaredLogger, db *gorm.DB, opts SyncOptions, from, to time.Time) int {
	total := 0
	for _, w := range monthlyWindows(from, to) {
		select {
		case <-ctx.Done():
			return -1
		default:
		}

		days, err := FetchCommitContributions(ctx, log, opts.Token, opts.User, w[0], w[1])
		if err != nil {
			MetricSyncErrors.Inc()
			log.Errorw("fetch window failed", "from", w[0], "to", w[1], zap.Error(err))
			continue
		}

		rows := make([]Contribution, 0, len(days))
		for d, c := range days {
			dt, perr := time.Parse("2006-01-02", d)
			if perr != nil {
				continue
			}
			rows = append(rows, Contribution{User: opts.User, Date: dt, Count: c})
			total += c
		}

		if err := Upsert(ctx, db, rows); err != nil {
			MetricSyncErrors.Inc()
			log.Errorw("upsert window failed", "from", w[0], "to", w[1], zap.Error(err))
			continue
		}
		MetricLastSync.SetToCurrentTime()
	}
	log.Infow("sync range complete", "from", from, "to", to, "commits", total)
	return total
}

// monthlyWindows splits [from,to) into <=1-month [start,end) pairs so the
// GraphQL contributions connection never needs pagination.
func monthlyWindows(from, to time.Time) [][2]time.Time {
	var out [][2]time.Time
	for start := from; start.Before(to); start = start.AddDate(0, 1, 0) {
		end := start.AddDate(0, 1, 0)
		if end.After(to) {
			end = to
		}
		out = append(out, [2]time.Time{start, end})
	}
	return out
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./code/ -v`
Expected: PASS (all three tests). `go build ./...` → succeeds.

- [ ] **Step 7: Commit**

```bash
git add code/metrics.go code/sync.go code/sync_test.go go.mod go.sum
git commit -m "feat: add prometheus metrics and background sync scheduler"
```

---

### Task 4: server rewrite, retire old path, tidy deps

**Files:**
- Modify (rewrite): `main.go`
- Delete: `code/commit.go`, `code/github.go`
- Modify: `go.mod`, `go.sum` (via `go mod tidy`)

**Interfaces:**
- Consumes: `code.ForAllTime`, `code.ForYear` (Task 1); `code.RunSync`, `code.SyncOptions` (Task 3); `promhttp`.
- Produces: HTTP endpoints `/healthz`, `/metrics`, `/data/contributions.csv`, `/data/{year}/weekly.csv`, `/` (static).

- [ ] **Step 1: Delete the retired files**

```bash
git rm code/commit.go code/github.go
```

- [ ] **Step 2: Rewrite `main.go`**

Replace the entire file with:
```go
package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/icco/code.natwelch.com/code"
	"github.com/icco/code.natwelch.com/static"
	"github.com/icco/gutil/etag"
	"github.com/icco/gutil/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

const (
	service = "code"
	project = "icco-cloud"
)

var log = logging.Must(logging.NewLogger(service))

func main() {
	port := envOr("PORT", "8080")
	user := envOr("GITHUB_USER", "icco")

	interval, err := time.ParseDuration(envOr("SYNC_INTERVAL", "6h"))
	if err != nil {
		log.Fatalw("invalid SYNC_INTERVAL", zap.Error(err))
	}
	startYear, err := strconv.Atoi(envOr("SYNC_START_YEAR", "2008"))
	if err != nil {
		log.Fatalw("invalid SYNC_START_YEAR", zap.Error(err))
	}

	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

	zgl := zapgorm2.New(log.Desugar())
	zgl.SetAsDefault()
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{Logger: zgl})
	if err != nil {
		log.Fatalw("cannot connect to database server", zap.Error(err))
	}
	if err := db.AutoMigrate(&code.Contribution{}); err != nil {
		log.Fatalw("cannot migrate Contribution", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go code.RunSync(ctx, log, db, code.SyncOptions{
		User:      user,
		Token:     os.Getenv("GITHUB_TOKEN"),
		Interval:  interval,
		StartYear: startYear,
	})

	srv := &http.Server{Addr: ":" + port, Handler: router(db, user)}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalw("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	stop()
	log.Infow("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Errorw("graceful shutdown failed", zap.Error(err))
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func router(db *gorm.DB, user string) http.Handler {
	r := chi.NewRouter()
	r.Use(etag.Handler(false))
	r.Use(middleware.RealIP)
	r.Use(logging.Middleware(log.Desugar(), project))

	crs := cors.New(cors.Options{
		AllowCredentials: true,
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		MaxAge:           300,
	})
	r.Use(crs.Handler)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hi."))
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/data/contributions.csv", func(w http.ResponseWriter, r *http.Request) {
		data, err := code.ForAllTime(r.Context(), db, user)
		if err != nil {
			log.Errorw("could not get contributions", zap.Error(err))
			http.Error(w, "could not get contributions", http.StatusInternalServerError)
			return
		}
		writeCSV(w, "date", data)
	})

	r.Get("/data/{year}/weekly.csv", func(w http.ResponseWriter, r *http.Request) {
		year, err := strconv.Atoi(chi.URLParam(r, "year"))
		if err != nil {
			http.Error(w, "could not parse year", http.StatusBadRequest)
			return
		}
		data, err := code.ForYear(r.Context(), db, user, year)
		if err != nil {
			log.Errorw("could not get weekly contributions", zap.Error(err))
			http.Error(w, "could not get weekly contributions", http.StatusInternalServerError)
			return
		}
		writeCSV(w, "week", data)
	})

	r.Mount("/", http.FileServer(http.FS(static.Assets)))
	return r
}

// writeCSV emits a sorted "<keyCol>,count" CSV.
func writeCSV(w http.ResponseWriter, keyCol string, data map[string]int64) {
	w.Header().Set("content-type", "text/csv")
	records := make([][]string, 0, len(data))
	for k, v := range data {
		records = append(records, []string{k, strconv.FormatInt(v, 10)})
	}
	sort.Slice(records, func(i, j int) bool { return records[i][0] < records[j][0] })

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{keyCol, "count"})
	if err := cw.WriteAll(records); err != nil {
		log.Errorw("error writing csv", zap.Error(err))
	}
}
```

- [ ] **Step 3: Tidy dependencies**

Run: `go mod tidy`
Expected: `go-github/v37` and `golang.org/x/oauth2` disappear from `go.mod`.
Verify: `grep -E "go-github|oauth2" go.mod` returns nothing.

- [ ] **Step 4: Build, vet, test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all succeed; no references to removed symbols.

- [ ] **Step 5: Smoke test against a local Postgres**

```bash
docker run -d --rm --name code-pg -e POSTGRES_PASSWORD=pw -e POSTGRES_DB=code -p 5433:5432 postgres:16
sleep 4
DATABASE_URL="postgresql://postgres:pw@localhost:5433/code?sslmode=disable" \
  GITHUB_TOKEN="" PORT=8080 go run . &
sleep 3
curl -sf localhost:8080/healthz            # -> hi.
curl -sf localhost:8080/metrics | grep code_   # -> custom metrics present
curl -sf localhost:8080/data/contributions.csv # -> "date,count" header, empty body
kill %1; docker stop code-pg
```
Expected: `/healthz` returns `hi.`, `/metrics` lists `code_last_sync_timestamp_seconds` etc., CSV returns just the header (empty DB, sync disabled with empty token). If a real `GITHUB_TOKEN` is exported, rows populate — verify the CSV has data lines.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: GraphQL-backed server, retire /save and REST commit path"
```

---

### Task 5: self-rendered contribution chart

**Files:**
- Modify (rewrite): `static/index.html`
- Create: `static/app.js`

`static/static.go` already embeds `*`, so `app.js` ships automatically.

**Interfaces:**
- Consumes: `GET /data/contributions.csv` (`date,count`).
- Produces: rendered inline-SVG calendar heatmap; no external scripts.

- [ ] **Step 1: Rewrite `static/index.html`**
```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>Nat's Code History</title>
    <link rel="stylesheet" href="https://unpkg.com/tachyons@4.12.0/css/tachyons.min.css" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
      .cell { shape-rendering: crispEdges; }
      .yr { font: 12px system-ui, sans-serif; fill: #57606a; }
    </style>
  </head>
  <body>
    <article class="pa3 pa5-ns">
      <h1 class="f3 f1-m f-headline-l">Nat's Code</h1>
      <p class="f6 measure">Daily commit contributions across public GitHub repositories, refreshed from GitHub's API.</p>
      <div id="chart" class="mt3">Loading…</div>
    </article>
    <footer class="black-80 pa3 pa5-ns">
      <p class="f6">
        <span class="dib">A site by <a class="link black-90 hover-light-blue" href="https://natwelch.com">@icco</a></span> /
        <a class="link black-90 hover-light-blue" href="https://github.com/icco/code.natwelch.com">Source</a> /
        <a class="link black-90 hover-light-blue" href="/data/contributions.csv">Data</a>
      </p>
    </footer>
    <script type="module" src="/app.js"></script>
  </body>
</html>
```

- [ ] **Step 2: Create `static/app.js`**
```js
// Fetch daily commit counts and render one GitHub-style heatmap per year.
const CELL = 12, GAP = 2, PAD = 24, WEEK = CELL + GAP;
const COLORS = ["#ebedf0", "#9be9a8", "#40c463", "#30a14e", "#216e39"];

const bucket = (n, max) => {
  if (n <= 0) return 0;
  if (max <= 4) return Math.min(n, 4);
  return Math.min(4, 1 + Math.floor((n - 1) / (max / 4)));
};

const dayOfWeek = (d) => (d.getUTCDay() + 6) % 7; // Mon=0 … Sun=6

function parseCSV(text) {
  const counts = new Map();
  for (const line of text.trim().split("\n").slice(1)) {
    const [date, count] = line.split(",");
    if (date) counts.set(date, Number(count) || 0);
  }
  return counts;
}

function yearSVG(year, counts) {
  const start = new Date(Date.UTC(year, 0, 1));
  const end = new Date(Date.UTC(year, 11, 31));
  let max = 1;
  for (const [d, c] of counts) if (d.startsWith(String(year))) max = Math.max(max, c);

  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  const weeks = 53;
  svg.setAttribute("width", PAD + weeks * WEEK);
  svg.setAttribute("height", PAD + 7 * WEEK + 8);
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", `Commit contributions for ${year}`);

  const label = document.createElementNS(svg.namespaceURI, "text");
  label.setAttribute("x", PAD);
  label.setAttribute("y", 14);
  label.setAttribute("class", "yr");
  label.textContent = year;
  svg.appendChild(label);

  const firstCol = new Date(Date.UTC(year, 0, 1 - dayOfWeek(start)));
  for (let d = new Date(firstCol); d <= end; d.setUTCDate(d.getUTCDate() + 1)) {
    if (d < start) continue;
    const key = d.toISOString().slice(0, 10);
    const col = Math.floor((d - firstCol) / 86400000 / 7);
    const row = dayOfWeek(d);
    const n = counts.get(key) || 0;
    const rect = document.createElementNS(svg.namespaceURI, "rect");
    rect.setAttribute("class", "cell");
    rect.setAttribute("x", PAD + col * WEEK);
    rect.setAttribute("y", PAD - 8 + row * WEEK);
    rect.setAttribute("width", CELL);
    rect.setAttribute("height", CELL);
    rect.setAttribute("rx", 2);
    rect.setAttribute("fill", COLORS[bucket(n, max)]);
    const title = document.createElementNS(svg.namespaceURI, "title");
    title.textContent = `${key}: ${n} commit${n === 1 ? "" : "s"}`;
    rect.appendChild(title);
    svg.appendChild(rect);
  }
  return svg;
}

async function main() {
  const chart = document.getElementById("chart");
  try {
    const res = await fetch("/data/contributions.csv");
    const counts = parseCSV(await res.text());
    if (counts.size === 0) {
      chart.textContent = "No contribution data yet — the sync is still populating.";
      return;
    }
    const years = [...new Set([...counts.keys()].map((d) => Number(d.slice(0, 4))))].sort((a, b) => b - a);
    chart.innerHTML = "";
    for (const y of years) {
      const wrap = document.createElement("div");
      wrap.className = "mb3 overflow-x-auto";
      wrap.appendChild(yearSVG(y, counts));
      chart.appendChild(wrap);
    }
  } catch (e) {
    chart.textContent = "Failed to load contribution data.";
  }
}

main();
```

- [ ] **Step 3: Verify it renders**

```bash
docker run -d --rm --name code-pg -e POSTGRES_PASSWORD=pw -e POSTGRES_DB=code -p 5433:5432 postgres:16
sleep 4
DATABASE_URL="postgresql://postgres:pw@localhost:5433/code?sslmode=disable" GITHUB_TOKEN="" go run . &
sleep 3
curl -sf localhost:8080/ | grep -q 'app.js' && echo "index ok"
curl -sf localhost:8080/app.js | grep -q 'yearSVG' && echo "app.js served"
kill %1; docker stop code-pg
```
Expected: both `echo`s fire. Optionally load `http://localhost:8080/` in a browser (or Playwright) and confirm the "No contribution data yet" message renders (empty DB), or the heatmap if a token was set. Confirm **no** request goes to `observablehq.com`.

- [ ] **Step 4: Commit**

```bash
git add static/index.html static/app.js
git commit -m "feat: self-render contribution heatmap, drop Observable embeds"
```

---

### Task 6: modernize + Dockerfile + Taskfile

**Files:**
- Modify (rewrite): `Dockerfile`
- Create: `Taskfile.yml`
- Delete: `.travis.yml`
- Modify: `go.mod` (`go 1.26.0`), `go.sum`

- [ ] **Step 1: Bump the module Go version and update deps**

```bash
go mod edit -go=1.26.0
go get -u ./...
go mod tidy
go build ./... && go vet ./... && go test ./...
```
Expected: build/vet/test all pass after updates. If any updated dep breaks the build, pin it back to the prior version (`go get module@vX.Y.Z`) and note it.

- [ ] **Step 2: Create `Taskfile.yml`**
```yaml
version: "3"

tasks:
  build:
    desc: Build the binary to bin/code
    cmds:
      - go build -o bin/code .
  run:
    desc: Run the server locally
    cmds:
      - go run .
  test:
    desc: Run tests
    cmds:
      - go test -v -cover ./...
  lint:
    desc: Vet and staticcheck
    cmds:
      - go vet ./...
      - go run honnef.co/go/tools/cmd/staticcheck@latest ./...
```

- [ ] **Step 3: Rewrite `Dockerfile`**
```dockerfile
FROM golang:1.26-bookworm AS builder

RUN go install github.com/go-task/task/v3/cmd/task@latest
RUN apt-get update && apt-get install -y git && apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN task build

FROM debian:bookworm-slim

LABEL org.opencontainers.image.source=https://github.com/icco/code.natwelch.com
LABEL org.opencontainers.image.description="Self-hosted GitHub commit-contribution history for @icco: in-process GraphQL sync into Postgres, self-rendered heatmap."
LABEL org.opencontainers.image.licenses=MIT

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

RUN groupadd -r app && useradd -r -u 1001 -g app app

WORKDIR /app
COPY --from=builder --chown=app:app /app/bin/ /app/bin/
USER app

EXPOSE 8080
CMD ["/app/bin/code"]
```

- [ ] **Step 4: Delete Travis**

```bash
git rm .travis.yml
```

- [ ] **Step 5: Verify the Docker build**

Run: `docker build -t code-test .`
Expected: build succeeds; final image runs `/app/bin/code`.
(If Docker is unavailable in the exec environment, run `task build` and confirm `bin/code` exists instead, and note Docker build must be verified in CI.)

- [ ] **Step 6: Commit**

```bash
git add Dockerfile Taskfile.yml go.mod go.sum
git rm --cached .travis.yml 2>/dev/null; true
git commit -m "chore: Go 1.26, multi-stage Dockerfile, Taskfile; drop Travis"
```

---

### Task 7: CI — build/push image + tests

**Files:**
- Create: `.github/workflows/docker.yml`, `.github/workflows/test.yml`

- [ ] **Step 1: Create `.github/workflows/docker.yml`**
```yaml
name: Create and publish Docker image
on:
  push:
    branches:
      - main
  pull_request:
env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}
jobs:
  build-and-push-image:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
      attestations: write
      id-token: write
    steps:
      - name: Checkout repository
        uses: actions/checkout@v7
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v4
      - name: Log in to the Container registry
        uses: docker/login-action@v4
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Extract metadata (tags, labels) for Docker
        id: meta
        uses: docker/metadata-action@v6
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
      - name: Build and push Docker image
        id: push
        uses: docker/build-push-action@v7
        with:
          context: .
          push: ${{ github.ref == 'refs/heads/main' }}
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
      - name: Generate artifact attestation
        if: github.ref == 'refs/heads/main'
        uses: actions/attest-build-provenance@v4
        with:
          subject-name: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          subject-digest: ${{ steps.push.outputs.digest }}
          push-to-registry: true
```
> Note: `docker/metadata-action` emits a `main` tag for pushes to the `main` branch, yielding `ghcr.io/icco/code.natwelch.com:main` (the deploy contract).

- [ ] **Step 2: Create `.github/workflows/test.yml`**
```yaml
name: test
on:
  push:
    branches:
      - main
  pull_request:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
      - run: go build ./...
      - run: go vet ./...
      - run: go test -v -cover ./...
```

- [ ] **Step 3: Validate workflow YAML**

Run: `python3 -c "import yaml,glob; [yaml.safe_load(open(f)) for f in glob.glob('.github/workflows/*.yml')]" && echo ok`
Expected: `ok` (valid YAML).

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/docker.yml .github/workflows/test.yml
git commit -m "ci: publish ghcr image on main; run tests on push/PR"
```

---

### Task 8: README + operator runbook

**Files:**
- Modify (rewrite): `README.md`
- Create: `docs/deploy.md`

- [ ] **Step 1: Rewrite `README.md`**
```markdown
# code.natwelch.com

[![Go Reference](https://pkg.go.dev/badge/github.com/icco/code.natwelch.com.svg)](https://pkg.go.dev/github.com/icco/code.natwelch.com)
[![Go Report Card](https://goreportcard.com/badge/github.com/icco/code.natwelch.com)](https://goreportcard.com/report/github.com/icco/code.natwelch.com)

Self-hosted view of [@icco](https://github.com/icco)'s GitHub commit history over time.

The service owns its own data pipeline: a background goroutine pulls **commit
contributions** from GitHub's GraphQL API (`contributionsCollection`), backfills
all history on boot, then refreshes on an interval, upserting daily counts into
Postgres (schema self-migrates via GORM `AutoMigrate`). The homepage renders an
inline-SVG calendar heatmap from `/data/contributions.csv` — no external
notebook or CDN. (This replaces the archived `github.com/icco/cron` scraper.)

## Endpoints

- `/` — contribution heatmap
- `/data/contributions.csv` — all-time daily counts (`date,count`)
- `/data/{year}/weekly.csv` — one year grouped by ISO week (`week,count`)
- `/healthz` — liveness
- `/metrics` — Prometheus

## Configuration

| Env | Default | Purpose |
|-----|---------|---------|
| `DATABASE_URL` | — | Postgres DSN |
| `GITHUB_TOKEN` | — | PAT for GraphQL; sync disabled if empty |
| `GITHUB_USER` | `icco` | user whose contributions to track |
| `SYNC_INTERVAL` | `6h` | refresh cadence |
| `SYNC_START_YEAR` | `2008` | earliest backfill year |
| `PORT` | `8080` | listen port |

## Develop

```
task build   # -> bin/code
task test
task run     # needs DATABASE_URL
```

Deploy notes (rope DB, PAT scopes) live in [docs/deploy.md](docs/deploy.md).
The image `ghcr.io/icco/code.natwelch.com:main` is published by CI on push to `main`.
```

- [ ] **Step 2: Create `docs/deploy.md`**
```markdown
# Deploy / operator runbook

The container is defined in `icco.me` (`mist/docker-compose.yml`, service
`code`) and served by caddy at `code.natwelch.com`. This repo publishes the
image; the steps below are the host-side wiring — **do not edit `icco.me` from
this repo's automation.**

## 1. Create the `code` database on rope

The compose `DATABASE_URL` points at `rope.local:5432/code` (user `nat`).
Create the database against rope's Postgres:

```bash
# On rope (adjust container name to the running postgres service):
docker exec -it postgres createdb -U nat code
# or, connecting from a host with psql:
psql "postgresql://nat:<password>@rope.local:5432/postgres" -c 'CREATE DATABASE code OWNER nat;'
```

The app runs `AutoMigrate` on boot, so no manual schema step is needed.

## 2. Provide a GitHub PAT

`GITHUB_TOKEN` in the `code` service is a placeholder (`REPLACE_WITH_GH_PAT`).
Create a token and set it on the host:

- **Public** commit contributions only: classic PAT with `read:user`
  (fine-grained: read-only "Profile"/account permissions).
- Include **private**-repo contributions: add `repo`, and enable
  *Settings → Profile → "Include private contributions on my profile"* on GitHub.

Set the real value in `icco.me/mist/docker-compose.yml` (or a host env/secret),
then `docker compose up -d code`. Do not commit the real token.

## 3. Security note (action item, not in this repo)

The shared Postgres password is currently committed in
`icco.me/mist/docker-compose.yml` (a real value, not a placeholder). Recommend
rotating it and moving it to a host secret / env file rather than the tracked
compose. This repo intentionally does not modify `icco.me`.

## 4. Verify

```bash
curl -sf https://code.natwelch.com/healthz         # hi.
curl -sf https://code.natwelch.com/metrics | grep code_
curl -sf https://code.natwelch.com/data/contributions.csv | head
```
`code_last_sync_timestamp_seconds` should advance after the first sync window;
the CSV gains rows once backfill runs (minutes, given the token).
```

- [ ] **Step 3: Commit**

```bash
git add README.md docs/deploy.md
git commit -m "docs: rewrite README, add deploy runbook"
```

---

## Self-Review

**Spec coverage:**
- Replace archived `cron` (task 1) → Tasks 2–4 (GraphQL client + in-process sync own the pipeline; `/save` retired).
- DB setup + `AutoMigrate` (task 2) → `AutoMigrate` in Task 4; DB creation in Task 8 runbook.
- Ship it — GHA build/push, delete Travis (task 3) → Task 7 + Task 6.
- Modernize Go + deps (task 4) → Task 6.
- Secrets (task 5) → Task 8 runbook (+ security flag on committed DB password).
- Viz story + `/metrics` (task 6) → Task 5 (self-render) + Task 3/4 (`/metrics`).

**Placeholder scan:** No TBD/TODO; every code step contains full code. The only "fill-in" values are intentional operator secrets in the runbook (`<password>`, PAT), which must not be committed.

**Type consistency:** `Contribution`, `Upsert`, `ForAllTime`, `ForYear`, `weekKey` (Task 1) are used verbatim in Tasks 3–4. `FetchCommitContributions(ctx, log, token, user, from, to)` (Task 2) is called with the same signature in Task 3. `SyncOptions`/`RunSync` (Task 3) match Task 4's call. CSV columns: `date,count` produced by Task 4, consumed by Task 5's `parseCSV`.

**Ordering:** Tasks 1–3 are purely additive (build stays green with the old `commit.go`/`github.go` present); Task 4 performs the atomic swap + `go mod tidy`. Every task ends green and committed.
