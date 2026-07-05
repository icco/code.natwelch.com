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
