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
