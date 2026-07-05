# Promote `code.natwelch.com` to a full-time service

**Date:** 2026-07-05
**Status:** Approved design → implementation plan

## Goal

Make `code.natwelch.com` a self-sufficient service that owns its own data
pipeline and publishes a deployable image. When done:

- CI publishes `ghcr.io/icco/code.natwelch.com:main` on push to `main`.
- The container boots, self-migrates its schema, and an in-process sync
  populates commit-contribution data directly from GitHub (no `icco/cron`).
- `code.natwelch.com` renders a live chart with no external-notebook
  dependency, and exposes `/healthz` and `/metrics`.

## Current state (verified)

- Go HTTP service (chi) on `:8080`; GORM against Postgres (`DATABASE_URL`).
- `main.go` **already** runs `db.AutoMigrate` on boot.
- `go.mod` **already** declares `go 1.25.0`; only the `Dockerfile`
  (`golang:1.21-alpine`) and the deleted-in-this-work `.travis.yml` are stale.
- Data source is broken: README points at the **archived** `github.com/icco/cron`
  scraper, which used to `POST /save` individual commits. Nothing populates data.
- Deploy contract (already wired in `icco.me`, must be matched, do **not** edit):
  - `icco.me/mist/docker-compose.yml` service `code`: image
    `ghcr.io/icco/code.natwelch.com:main`, one service, web binary on `:8080`,
    caddy label `code.natwelch.com`.
  - `DATABASE_URL` = `postgresql://nat:<pw>@rope.local:5432/code?sslmode=disable`
    (a real password is already committed there — see Security note).
  - `GITHUB_TOKEN` = `REPLACE_WITH_GH_PAT` (still a placeholder).

## Decisions (from brainstorming)

1. **Sync runs in-process.** The compose defines a single `code` service running
   the web binary, so the sync is a background goroutine in that binary — one
   image, one container, zero `icco.me` changes.
2. **Data source = GitHub GraphQL, commits only.** Use
   `user.contributionsCollection.commitContributionsByRepository`, aggregated to
   a daily commit count. (Not the all-types green-squares calendar.)
3. **Self-rendered viz.** Replace the Observable notebook embeds with a local
   inline-SVG chart reading the CSV endpoints. No external notebook, no CDN.
4. **Retire the push path.** Delete `POST /save`, `Commit{SHA}`, `CheckAndSave`,
   `UserRepos`. The GraphQL sync fully owns the pipeline.
5. **Host-side tasks are a runbook.** DB creation on `rope` and setting the real
   PAT are documented for the operator; not executed from this repo.

## Architecture

Single Go binary (`code`) that on startup:
1. Connects to Postgres, runs `AutoMigrate(&code.Contribution{})`.
2. Launches `go code.RunSync(ctx, ...)` — a background scheduler goroutine.
3. Serves HTTP on `:8080` until `SIGINT`/`SIGTERM`, which cancels `ctx` so the
   sync goroutine drains and exits cleanly.

```
main.go
 ├─ gorm.Open + AutoMigrate(Contribution)
 ├─ go code.RunSync(ctx, db, token, user, opts)   // ticker loop
 └─ http.ListenAndServe(":8080", router)
      /healthz                      → liveness
      /metrics                      → promhttp + custom gauges
      /data/contributions.csv       → all-time daily counts
      /data/{year}/weekly.csv       → one year, grouped by ISO week
      /                             → static site (self-rendered SVG)
```

### Components

**`code/contribution.go` — model + queries.** Replaces `commit.go`.
```go
type Contribution struct {
    ID        uint      `gorm:"primarykey"`
    User      string    `gorm:"index:idx_user_date,unique"`
    Date      time.Time `gorm:"index:idx_user_date,unique;type:date"`
    Count     int
    UpdatedAt time.Time
}
```
- `Upsert(ctx, db, rows)` — GORM `clause.OnConflict{Columns:{user,date}, DoUpdates: {count, updated_at}}` for idempotent writes.
- `ForAllTime(ctx, db, user) → map[string]int64` (day → count).
- `ForYear(ctx, db, user, year) → map[string]int64` (ISO-week label → summed count).

**`code/github.go` — GraphQL client.** Rewrite. Drops `go-github/v37` and
`oauth2`. Raw HTTPS POST to `https://api.github.com/graphql` with
`Authorization: bearer <token>`.
- `FetchCommitContributions(ctx, token, user, from, to time.Time) (map[string]int, error)`
  returns day → commit count for a range ≤ 1 year.
- Query shape:
  ```graphql
  query($login:String!, $from:DateTime!, $to:DateTime!) {
    user(login:$login) {
      contributionsCollection(from:$from, to:$to) {
        commitContributionsByRepository(maxRepositories:100) {
          contributions(first:100) {
            nodes { occurredAt commitCount }
            pageInfo { hasNextPage endCursor }
          }
        }
      }
    }
  }
  ```
- Sum `commitCount` per `occurredAt` day across repos; paginate `contributions`
  per repo on `hasNextPage`. If `maxRepositories:100` is hit, log a warning
  (no silent truncation).

**`code/sync.go` — scheduler.** In-process, `ctx`-cancellable.
- `RunSync(ctx, db, token, user, opts)`:
  - On start: **full backfill** — loop years `[StartYear, currentYear]`, one
    1-year GraphQL window each, upsert. (~20 cheap queries; each GraphQL call = 1
    rate-limit point.)
  - Then a `time.Ticker(Interval)`: refresh the **trailing window** (current +
    prior year, to catch late/backdated commits) and upsert.
  - `select` on `ctx.Done()` for clean shutdown; log per-run counts and errors.
- Config via env with defaults: `GITHUB_USER` (`icco`), `SYNC_INTERVAL` (`6h`),
  `SYNC_START_YEAR` (`2008`).

**`main.go` — server.** Keep chi setup, CORS, etag, logging, `/healthz`.
- Keep `AutoMigrate` (now `Contribution`).
- Remove `POST /save`.
- Rewrite `/data/contributions.csv` and `/data/{year}/weekly.csv` against the
  new model (columns `date,count` / `week,count`).
- Add `/metrics` (`promhttp.Handler()`) + custom collectors:
  `code_last_sync_timestamp_seconds`, `code_sync_errors_total`,
  `code_contributions_total`.
- Launch the sync goroutine; wire graceful shutdown.

**`static/` — self-rendered viz.** `index.html` + `app.js` (vanilla, no build).
- Fetch `/data/contributions.csv`; render an inline-SVG GitHub-style calendar
  heatmap (weeks × weekdays, color-scaled by count) plus a per-year totals bar.
- Keep tachyons for layout. Remove Observable `Runtime`/`define` scripts and the
  `observablehq.com/@icco` links.

### Build / ship / modernize

- **`Dockerfile`** — multi-stage matching `etu-backend`:
  `golang:1.26-bookworm` builder (installs `task`, builds via `task build` →
  `bin/code`); `debian:bookworm-slim` final with `ca-certificates`, non-root
  `app` user, OCI labels, `EXPOSE 8080`, `CMD ["/app/bin/code"]`.
- **`Taskfile.yml`** — `build` (`go build -o bin/code .`), `test`
  (`go test -v -cover ./...`), `lint` (`go vet` + staticcheck), `run`. Per
  personal-project convention (Taskfile, not Makefile).
- **`.github/workflows/docker.yml`** — build + push
  `ghcr.io/icco/code.natwelch.com:main` on push to `main` (metadata-action tags),
  matching `etu-backend`. Push only on `main`; build-only on PRs.
- **`.github/workflows/test.yml`** — `go test` + `go vet` on push/PR (minimal).
- **Delete `.travis.yml`.**
- **Modernize:** `Dockerfile` Go 1.26; bump `go.mod` `go 1.25.0` → `1.26.0`;
  `go get -u ./...`; verify `go build ./...` and `go vet ./...`.
- **README:** drop the `cron` + Observable references; document the in-repo
  GraphQL sync, env vars, self-rendered viz, and ghcr image.

### Tests

Table-driven, no live Postgres:
- `code/github_test.go` — `httptest` server returns a canned GraphQL payload;
  assert day→count aggregation across repos and pagination.
- `code/contribution_test.go` — day→ISO-week aggregation for `ForYear`.

### Runbook (`docs/deploy.md`) — operator steps I cannot execute

- **Create DB on `rope`:** exact command against rope's Postgres container to
  `CREATE DATABASE code` owned by `nat` (DB name/host/user already in the compose
  `DATABASE_URL`).
- **GITHUB_TOKEN PAT:** scope `read:user` for public commit contributions; add
  `repo` only if private-repo contributions should be included. Replace
  `REPLACE_WITH_GH_PAT` in `icco.me/mist/docker-compose.yml` (operator edits
  `icco.me`, not this repo).
- **Security note:** the shared Postgres password is committed in
  `icco.me/mist/docker-compose.yml:728` (real, not a placeholder). Flagged for
  rotation / migration to a managed secret. This work does not modify `icco.me`.

## Non-goals / YAGNI

- No `cmd/sync` standalone binary (in-process scheduler is sufficient for the
  single-service deploy; a manual backfill can run the same code locally).
- No second compose service; no `icco.me` edits.
- No REST per-repo commit crawl (GraphQL is far cheaper).
- No auth on `/data/*` or `/metrics` (public, read-only; matches current site).

## Done when

CI publishes `:main`; the container boots and self-migrates; the in-process
GraphQL sync populates daily commit counts (no `cron` repo); `/healthz` and
`/metrics` respond; and `code.natwelch.com` renders a live self-hosted chart.
