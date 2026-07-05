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

- **Public** only: classic PAT with `read:user`.
- **Private** repos: classic PAT with the `repo` scope, plus *Settings → Profile
  → "Include private contributions on my profile"*. Fine-grained tokens don't
  reliably itemize private repos here.

`/metrics` exports `code_private_commits_visible` — `0` when private commits are
being dropped (wrong token scope / profile setting off). Alert on `== 0`.

Provide the real value via a host env file / secret referenced by the compose
service (not committed) — e.g. a `.env` on the host or your secret manager —
then `docker compose up -d code`. Avoid pasting the real token directly into the
tracked `icco.me/mist/docker-compose.yml`; if you do edit that file, keep the
placeholder in git and inject the real value at deploy time. Never commit the
real token.

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
the CSV gains rows once the backfill runs (the first full backfill can take a
while depending on GitHub API rate limits).
