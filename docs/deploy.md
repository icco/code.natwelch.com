# Deploy runbook

Run `ghcr.io/icco/code.natwelch.com:main` behind a reverse proxy that serves `code.natwelch.com` and forwards to port 8080. Inject secrets from the host environment at deploy time and never commit them.

## 1. Create the database

```bash
psql "postgresql://alice@db.example:5432/postgres" -c 'CREATE DATABASE code OWNER alice;'
```

Point `DATABASE_URL` at it. The app runs `AutoMigrate` on boot, so there is no schema step.

## 2. Create a GitHub token

- Public contributions only: classic PAT with `read:user`.
- Private contributions too: classic PAT with the `repo` scope, plus Settings > Public profile > "Include private contributions on my profile". Fine-grained tokens don't reliably itemize private repos.

Set it as `GITHUB_TOKEN`. With no token the server still serves stored data but never syncs.

## 3. Verify

```bash
curl -sf https://code.natwelch.com/healthz          # hi.
curl -sf https://code.natwelch.com/metrics | grep code_
curl -sf https://code.natwelch.com/data/contributions.csv | head
```

| Metric | Meaning |
|--------|---------|
| `code_last_sync_timestamp_seconds` | last window written; should advance every `SYNC_INTERVAL` |
| `code_sync_errors_total` | windows that failed to fetch or upsert after retries |
| `code_contributions_total` | commits stored for the user |
| `code_private_commits_visible` | `0` when any window in the last sync pass had private activity GitHub didn't itemize (token scope or profile setting), and stays `0` until a pass fetches every window cleanly; alert on `== 0` |

The first backfill fetches one window per month since `SYNC_START_YEAR` and can take a while under GitHub's rate limits.
