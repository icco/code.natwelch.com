# code.natwelch.com

[![Go Reference](https://pkg.go.dev/badge/go.icco.me/code.natwelch.com.svg)](https://pkg.go.dev/go.icco.me/code.natwelch.com)

Self-hosted view of [@icco](https://github.com/icco)'s GitHub commit history over time, served at [code.natwelch.com](https://code.natwelch.com).

## Install

```
docker pull ghcr.io/icco/code.natwelch.com:main
```

CI publishes the image on every push to `main`.

## Usage

```
docker run -p 8080:8080 \
  -e DATABASE_URL=postgres://alice@db.example:5432/code \
  -e GITHUB_TOKEN=... \
  ghcr.io/icco/code.natwelch.com:main
```

| Env | Default | Purpose |
|-----|---------|---------|
| `DATABASE_URL` | none | Postgres DSN |
| `GITHUB_TOKEN` | none | PAT for GraphQL; sync is disabled if empty |
| `GITHUB_USER` | `icco` | user whose contributions to track |
| `SYNC_INTERVAL` | `6h` | refresh cadence |
| `SYNC_START_YEAR` | `2008` | earliest backfill year |
| `PORT` | `8080` | listen port |

Token scopes and deploy steps are in [docs/deploy.md](docs/deploy.md).

## Endpoints

- `/`: contribution heatmap, one calendar per year
- `/data/contributions.csv`: all-time daily counts (`date,count`)
- `/data/{year}/weekly.csv`: one year grouped by ISO week (`week,count`)
- `/healthz`: liveness
- `/metrics`: Prometheus

## How it works

A background goroutine pulls commit contributions from GitHub's GraphQL API ([`contributionsCollection`](https://docs.github.com/en/graphql/reference/objects#contributionscollection)) in calendar-month windows and upserts daily counts into Postgres. GORM `AutoMigrate` creates the schema on boot.

- On boot it backfills every year from `SYNC_START_YEAR` that has no rows yet.
- Every `SYNC_INTERVAL` it re-fetches the trailing 13 calendar months, writing zeros for days with no commits so removed commits drop out.
- The homepage renders an inline SVG heatmap from `/data/contributions.csv`, with no external scripts or CDN.

## Develop

```
task build   # -> bin/code
task test
task lint
task run     # needs DATABASE_URL
```
