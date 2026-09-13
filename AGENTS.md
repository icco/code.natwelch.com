# AGENTS.md

Guidance for coding agents working on code.natwelch.com.

## Project Overview

A source code browser and vanity redirect server written in Go (`github.com/icco/code.natwelch.com`).

## Commands (Taskfile)

Run via `task <name>`:
- `task build` — Build server binary to `bin/code`
- `task run` — Run server locally
- `task test` — Run tests with coverage (`go test -v -cover ./...`)
- `task lint` — Run `go vet` and `staticcheck`

## Architecture & Layout

- `main.go` — Entrypoint and HTTP routing.
- `lib/` — Repository inspection and template handlers.
- `templates/` — HTML templates.

## Conventions

- Follow icco Go conventions (`github.com/icco/gutil` for logging and common helpers).
- PR titles and commits must follow Conventional Commits with lowercase subjects.
- Ensure `task lint` and `task test` pass before committing.
