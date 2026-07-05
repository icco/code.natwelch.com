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
