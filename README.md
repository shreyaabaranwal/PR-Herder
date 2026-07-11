<div align="center">

# PR Herder

**The Slack agent that tames the open-source pull-request firehose.**

Triage incoming PRs, route reviews to the right people, catch flaky CI, and approve or request changes — without ever leaving Slack.

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Slack](https://img.shields.io/badge/Slack-Block%20Kit-4A154B?logo=slack&logoColor=white)](https://api.slack.com/block-kit)
[![MCP](https://img.shields.io/badge/GitHub-MCP%20Server-000000?logo=github)](https://github.com/github/github-mcp-server)
[![Ollama](https://img.shields.io/badge/LLM-Ollama%20(local)-1a1a1a)](https://ollama.com)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-ready-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)

Built for the [Slack Agent Builder Challenge](https://slack-agent-builder-challenge.devpost.com/)

</div>

---

## Table of contents

- [Overview](#overview)
- [Why PR Herder](#why-pr-herder)
- [Architecture](#architecture)
- [Design principle](#design-principle)
- [Features by layer](#features-by-layer)
- [Tech stack](#tech-stack)
- [Getting started](#getting-started)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Observability](#observability)
- [Project structure](#project-structure)
- [Team](#team)
- [License](#license)

---

## Overview

PR Herder is a Go-based Slack agent that watches GitHub pull requests, triages them using deterministic rules, optionally summarizes ambiguous changes with a locally-hosted LLM, and posts an interactive card straight into Slack. Reviewers approve or request changes with a single click — the action executes on GitHub in real time through the official **GitHub MCP Server**, using JSON-RPC over Server-Sent Events.

No context-switching. No stale PR queues. No CI noise. Just a Slack channel that tells you exactly what needs your attention, and lets you act on it immediately.

## Why PR Herder

Open-source maintainers and busy engineering teams lose real time every day tabbing between GitHub and Slack — checking CI status, re-reading diffs to figure out who should review, and manually pinging people about stale PRs. PR Herder collapses that whole loop into one surface.

- **Deterministic-first triage** — routing and risk decisions are made by explicit rules, not by an LLM. This keeps behavior predictable, auditable, and fast.
- **AI where it actually helps** — a locally-hosted LLM (Ollama) is invoked *only* for large, unrouted diffs where a human would otherwise have to read the whole thing to understand intent.
- **Real actions, not suggestions** — Approve and Request Changes buttons in Slack execute genuine GitHub reviews via MCP, authorized against real repo permissions before anything happens.
- **Production-grade from day one** — retries with backoff, a dead-letter queue, Prometheus metrics, structured logging, health probes, and both Docker and Kubernetes deployment paths.

## Architecture

### Request flow

```mermaid
flowchart TD
    A["GitHub pull request<br/>opened or updated"] -->|webhook| B["Webhook ingest<br/>HMAC verified, queued in Postgres"]
    B --> C["Async worker<br/>polls queue every 5s"]
    C --> D{"Triage engine<br/>deterministic rules"}
    D -->|clear decision| F["Slack interactive card"]
    D -->|large, unrouted diff| E["Ollama LLM<br/>local PR summary"]
    E --> F
    F -->|Approve / Request changes| G["Authorization check<br/>Slack identity to GitHub permission"]
    G --> H["GitHub MCP executor<br/>JSON-RPC over SSE"]
    H --> I["GitHub review created"]

    style A fill:#E6F1FB,stroke:#185FA5,color:#042C53
    style I fill:#E6F1FB,stroke:#185FA5,color:#042C53
    style B fill:#F1EFE8,stroke:#5F5E5A,color:#2C2C2A
    style C fill:#F1EFE8,stroke:#5F5E5A,color:#2C2C2A
    style D fill:#EEEDFE,stroke:#534AB7,color:#26215C
    style E fill:#FAEEDA,stroke:#854F0B,color:#412402
    style F fill:#FBEAF0,stroke:#993556,color:#4B1528
    style G fill:#E1F5EE,stroke:#0F6E56,color:#04342C
    style H fill:#E1F5EE,stroke:#0F6E56,color:#04342C
```

### System layers

```mermaid
flowchart LR
    subgraph core["Core pipeline"]
        direction TB
        W["Webhook ingest"] --> T["Triage engine"] --> S["Slack publisher"]
    end

    subgraph ai["AI layer"]
        O["Ollama local LLM summary"]
    end

    subgraph ops["Operational layers"]
        direction TB
        Sc["Scheduler: stale-PR digest, quiet hours"]
        Hd["Hardening: retries, DLQ, metrics, Docker, K8s"]
    end

    core -->|ambiguous PRs only| ai
    core -.-> ops

    style core fill:#EEEDFE,stroke:#534AB7,color:#26215C
    style ai fill:#FAEEDA,stroke:#854F0B,color:#412402
    style ops fill:#F1EFE8,stroke:#5F5E5A,color:#2C2C2A
```

## Design principle

> PR Herder follows a **deterministic-first design**: rule-based triage always makes the primary decision. The LLM is only ever asked to summarize a PR the rules couldn't confidently classify — never to route, approve, or decide anything security-relevant. This keeps the system predictable and auditable while still benefiting from AI assistance exactly where it adds value.

This is documented formally as ADR 0001 in the codebase and enforced structurally: the triage engine (`internal/triage`) is a pure, network-free function. The LLM call happens strictly *after* triage, only when `Result.Ambiguous == true`, and its output is attached as a supplementary summary — it never overrides or feeds back into the routing decision.

## Features by layer

| Layer | What it does | Status |
|---|---|---|
| 0 — Foundation | Domain model, config loading, Postgres schema | Complete |
| 1 — Webhook ingest | HMAC-verified, idempotent event storage | Complete |
| 2 — Triage engine | Deterministic size/path/contributor rules | Complete |
| 3 — Slack publisher | Block Kit triage cards | Complete |
| 4 — Slack interactivity | Signature-verified button actions | Complete |
| 5 — GitHub MCP | JSON-RPC client over SSE, real GitHub actions | Complete |
| 6 — Flaky CI detection | Statistical classifier over check-run history | Complete |
| 7 — LLM summary | Local Ollama backend, ambiguous PRs only | Complete |
| 8 — Scheduler | Daily stale-PR digest, quiet hours | Complete |
| 9 — Production hardening | Retries, DLQ, metrics, health probes, Docker, Kubernetes | Complete |

## Tech stack

- **Language:** Go 1.25
- **Database:** PostgreSQL 16
- **Messaging:** Slack Block Kit + Events API
- **Integration:** GitHub MCP Server (JSON-RPC 2.0 over Server-Sent Events)
- **AI:** Ollama (local inference, `llama3.2:3b`)
- **Observability:** Prometheus metrics, structured JSON logging (`slog`)
- **Deployment:** Docker (multi-stage, distroless), Kubernetes manifests

## Getting started

### Prerequisites

- Go 1.25+
- Docker
- [Ollama](https://ollama.com) with a model pulled (`ollama pull llama3.2:3b`)
- A Slack app with a bot token and signing secret
- A GitHub personal access token with `repo` scope

### Local setup

```bash
git clone https://github.com/shreyaabaranwal/PR-Herder.git
cd PR-Herder
cp .env.example .env   # fill in your real values

docker run -d --name prherder-pg \
  -e POSTGRES_USER=prherder -e POSTGRES_PASSWORD=prherder -e POSTGRES_DB=prherder \
  -p 5432:5432 postgres:16

for f in migrations/*.sql; do
  psql postgres://prherder:prherder@localhost:5432/prherder -f "$f"
done

go run cmd/prherder/main.go
```

Expose it to GitHub with a tunnel (e.g. `ngrok http 8090`) and point your repo's webhook at `<tunnel-url>/github/webhook`.

## Configuration

All configuration is environment-driven — see `.env.example` for the full list. Key variables:

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Postgres connection string |
| `SLACK_BOT_TOKEN` / `SLACK_SIGNING_SECRET` | Slack app credentials |
| `GITHUB_WEBHOOK_SECRET` / `GITHUB_READ_TOKEN` | GitHub webhook + API access |
| `GITHUB_MCP_URL` | GitHub MCP server endpoint |
| `OLLAMA_URL` / `OLLAMA_MODEL` | Local LLM backend |
| `STALE_PR_DAYS` / `DIGEST_HOUR_LOCAL` / `QUIET_HOURS` | Scheduler behavior |

## Deployment

### Docker

```bash
docker build -t prherder:latest .
docker run --env-file .env -p 8090:8090 prherder:latest
```

### Kubernetes

```bash
kubectl apply -f k8s/namespace.yaml
kubectl create secret generic prherder-secrets --namespace=pr-herder --from-env-file=.env
kubectl apply -f k8s/configmap.yaml -f k8s/postgres.yaml -f k8s/deployment.yaml -f k8s/service.yaml
```

Liveness and readiness probes are wired to `/livez` and `/readyz`; Prometheus can scrape `/metrics` directly off the pod (annotations already included in `k8s/deployment.yaml`).

## Observability

- **Health:** `/healthz`, `/livez`, `/readyz` (readiness checks live Postgres connectivity)
- **Metrics:** `/metrics` — Prometheus format, covering webhook throughput, triage duration, LLM outcomes, Slack publish outcomes, worker errors, and dead-letter counts
- **Logging:** structured JSON via `slog`, with `delivery_id` correlation across the full request lifecycle
- **Reliability:** exponential-backoff retries on LLM/Slack calls, plus a cross-cycle dead-letter queue so a permanently broken event stops retrying after 5 attempts instead of looping forever

## Project structure

```
cmd/prherder/         entrypoint, wires every layer together
internal/
  domain/              canonical PR model, decoupled from GitHub's wire format
  ingest/              webhook handler + async worker
  triage/              deterministic rule engine
  llm/                 Gemini + Ollama backends behind one interface
  slackui/             Block Kit publisher + interaction handler
  githubmcp/           MCP client, JSON-RPC over SSE
  authz/               permission checks before any GitHub write
  scheduler/           stale-PR digest, quiet hours
  store/               all SQL lives here
  metrics/             Prometheus counters and histograms
  httpmw/              panic-recovery middleware
migrations/            versioned schema, applied in order
k8s/                   Kubernetes manifests
```


<div align="center">

## Team : Vitamin She

</div>


## License

See [LICENSE](./LICENSE).

---

<div align="center">

**PR Herder** — never leave Slack to ship a review.

</div>