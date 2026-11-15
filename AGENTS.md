# AGENTS.md

Guidance for coding agents working on code.natwelch.com.

## Project Overview

A Go service (`go.icco.me/code.natwelch.com`) that syncs a GitHub user's daily commit contributions into Postgres and renders them as a heatmap. See [README.md](README.md) for configuration and endpoints.

## Commands (Taskfile)

- `task build`: build the server binary to `bin/code`
- `task run`: run the server locally (needs `DATABASE_URL`)
- `task test`: run tests with coverage
- `task lint`: run `go vet` and `staticcheck`

## Layout

- `main.go`: config, chi router, CSV endpoints, graceful shutdown.
- `code/contribution.go`: `Contribution` model and Postgres queries.
- `code/graphql.go`: GitHub GraphQL client for one month of contributions.
- `code/sync.go`: backfill and periodic refresh loop.
- `code/metrics.go`: Prometheus gauges and counters.
- `static/`: embedded `index.html` and `app.js` that draw the heatmap.
- `docs/deploy.md`: operator runbook.

## Conventions

- Use `go.icco.me/gutil` for logging and ETags, chi for routing, and GORM `AutoMigrate` for schema.
- PR titles and commits follow Conventional Commits with lowercase subjects.
- `task lint` and `task test` must pass before committing.
