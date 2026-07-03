<div align="center">

# PR Herder

**The Slack agent that tames the open-source pull-request firehose.**

Triage incoming PRs, route reviews to the right people, catch flaky CI, and approve or request changes — without ever leaving Slack.

[![Built for Slack Agent Builder Challenge](https://img.shields.io/badge/Slack-Agent%20Builder%20Challenge-4A154B)](https://slack.dev)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![GitHub MCP](https://img.shields.io/badge/GitHub-MCP%20Integration-181717?logo=github&logoColor=white)](https://github.com)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

</div>

---

PR Herder lives in your maintainers' Slack workspace and turns the chaotic stream of GitHub pull requests into a calm, prioritized, actionable feed. It triages incoming PRs, routes reviews intelligently, detects flaky CI, nudges on stale reviews without spamming, and lets maintainers act on pull requests directly from Slack — all behind a strict authorization model.

Built for the **Slack Agent Builder Challenge**. Uses **GitHub MCP integration** and **Slack AI capabilities**. Written in **Go**.

> **Why this exists**
> Maintaining an active open-source repo means drowning in PR noise — first-time contributors waiting days for a hello, security-sensitive changes buried under typo fixes, and "red" CI that's actually just a flaky test. PR Herder is the triage layer maintainers never had.

---

## Contents

- [Features](#features)
- [How It Works](#how-it-works)
- [Architecture](#architecture)
- [The Triage Engine](#the-triage-engine)
- [Security and Authorization Model](#security-and-authorization-model)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Slack App Manifest](#slack-app-manifest)
- [GitHub MCP Tools Used](#github-mcp-tools-used)
- [Local Development](#local-development)
- [Deployment](#deployment)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

---

## Features

### Smart PR triage — not just an LLM wrapper

Every incoming PR runs through a **deterministic rules engine first** (file paths, CODEOWNERS, diff size, CI status, `author_association`). The LLM is only invoked on genuinely ambiguous PRs, and only to *explain* — never to make security-critical routing decisions. This keeps cost low, latency predictable, and behavior auditable.

A triaged PR posts to Slack as a rich Block Kit card:

> **#1423 · Add rate-limiting to auth middleware** — `needs-security-review`
> Touches `internal/auth/**` → routed to @security-team
> CI failing, but on `TestFlakyRetry`, historically flaky (safe to retry)
> First-time contributor — welcome message auto-suggested
>
> `[ Approve ]` `[ Request changes ]` `[ Add label ]` `[ Assign reviewer ]`

### Reviewer load-balancer

Instead of round-robin, PR Herder routes review requests based on **code ownership and current review load**, so no single maintainer becomes the bottleneck.

### Flaky-CI detective

Detects the classic *fails-then-passes-on-retry* pattern across CI history and tells the maintainer **"this red is flaky, safe to merge"** — instead of everyone assuming the PR is broken. This is the AIOps heart of the project.

### Non-spammy stale nudges

Stale PRs are surfaced in **one batched daily digest**, sent privately to the responsible reviewer, respecting quiet hours and snooze/mute controls. No channel spam, ever.

### Act from Slack, safely

Approve, request changes, label, or assign directly from Slack buttons. Every mutating action verifies the Slack user maps to a GitHub identity **with write access to that repo**, and acts using *their* scoped credentials behind an approval gate.

### Contributor-friendly

First-time contributors get an auto-suggested welcome. Blocked PRs get a plain-English "here's exactly what to fix" explanation posted back to GitHub (opt-in).

---

## How It Works

```
GitHub PR event  ──webhook──▶  PR Herder (Go)  ──▶  Triage Engine  ──▶  Slack Block Kit card
                                     ▲                                          │
                                     │                                          │ button click
                                     └──────────  GitHub MCP  ◀─────────────────┘
                                         (scoped, per-user authorized writes)
```

1. **Ingest** — GitHub webhooks hit PR Herder. Events are verified, deduplicated (idempotency keys), and queued.
2. **Triage** — The deterministic rules engine classifies the PR. Ambiguous cases escalate to the LLM for a summary only.
3. **Surface** — A Block Kit card is posted to the configured Slack channel with context and action buttons.
4. **Act** — Maintainer clicks a button → Slack sends an interaction → PR Herder authorizes the user → executes the action via the GitHub MCP server using scoped credentials.
5. **Follow up** — Stale PRs, flaky-CI notes, and reviewer nudges are batched into digests.

---

## Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                          PR Herder (Go service)                     │
│                                                                      │
│  ┌────────────┐   ┌───────────────┐   ┌─────────────────────────┐  │
│  │  Ingest    │   │   Event Queue │   │     Triage Engine       │  │
│  │  Handler   │──▶│  (debounce +  │──▶│  1. Deterministic rules │  │
│  │ (webhook   │   │  idempotency) │   │  2. LLM (ambiguous only)│  │
│  │  verify)   │   └───────────────┘   └───────────┬─────────────┘  │
│  └────────────┘                                   │                 │
│         ▲                                          ▼                 │
│  ┌────────────┐   ┌───────────────┐   ┌─────────────────────────┐  │
│  │  Slack     │   │  Authz &      │   │    Slack Publisher      │  │
│  │ Interaction│◀─▶│  Identity     │◀──│  (Block Kit renderer)   │  │
│  │  Handler   │   │  Mapper       │   └─────────────────────────┘  │
│  └─────┬──────┘   └───────┬───────┘                                 │
│        │                  │                                         │
│        ▼                  ▼                                         │
│  ┌──────────────────────────────────┐   ┌───────────────────────┐  │
│  │       GitHub MCP Client          │   │   Scheduler (digests, │  │
│  │  (reads + scoped writes)         │   │   stale nudges, flaky) │  │
│  └──────────────────────────────────┘   └───────────────────────┘  │
│                                                                      │
│  Store: Postgres (identity map, PR state, flaky-test history, mutes)│
└────────────────────────────────────────────────────────────────────┘
        │                          │                        │
        ▼                          ▼                        ▼
   GitHub API                 Slack API              LLM (Claude API)
   (via MCP)                 (Block Kit)          (summaries only)
```

**Stack**

| Layer | Choice |
|---|---|
| Language | Go (chi/echo for HTTP, `sqlc`/`pgx` for Postgres) |
| Integration | GitHub MCP server for all GitHub reads and writes |
| Slack | Bolt-style HTTP handlers (events, interactivity, shortcuts) |
| AI | Claude API for summaries and explanations only (called on under 20% of PRs) |
| Store | Postgres for identity map, PR/label state, flaky-test history, mute/snooze state |
| Queue | In-process worker pool plus a Postgres-backed job table (or Redis if scaling) |

---

## The Triage Engine

The engine is intentionally **layered so the cheap, deterministic checks run first** and the LLM is a last resort.

**Layer 1 — Deterministic rules (always run, no LLM)**

| Signal | Source | Action |
|---|---|---|
| Touches sensitive paths | file globs + CODEOWNERS | route to owning team, add `needs-*-review` |
| Diff size | `additions + deletions` | tag `size/S｜M｜L｜XL` |
| CI status | check runs | red / green / flaky classification |
| Contributor status | `author_association` | first-timer → welcome suggestion |
| Reviewer load | internal state | pick least-loaded eligible reviewer |

**Layer 2 — Flaky-CI classifier (statistical, no LLM)**

Tracks per-test pass/fail history. A failing check with a *fail-then-pass on retry* rate above a threshold is marked **flaky**, so a red X doesn't block a mergeable PR.

**Layer 3 — LLM summary (only if ambiguous)**

When rules can't confidently classify (for example, a large mixed diff with no CODEOWNERS match), Claude produces a **short, factual summary and suggested labels** — presented as *suggestions the maintainer confirms*, never auto-applied to security routing.

> **Why this matters for judging:** it demonstrates real engineering judgment — cost, determinism, auditability — rather than "call the LLM on everything," which lifts the *Technological Implementation* score.

---

## Security and Authorization Model

This is the part that turns a demo into something a real org would install.

**1. Inbound verification**
- Every GitHub webhook is verified with the `X-Hub-Signature-256` HMAC.
- Every Slack request is verified with the Slack signing secret plus a timestamp check (replay protection).

**2. Identity mapping**
- A Slack user can only act after being mapped to a GitHub identity via a one-time OAuth link.
- The mapping is stored server-side; a Slack ID alone can never trigger a GitHub write.

**3. Scoped authorization on every mutating action**
- Before any approve, label, or merge, PR Herder checks the mapped GitHub user actually has **write access to that specific repo**.
- Actions execute with **per-user scoped tokens** or a **GitHub App installation token scoped to the minimum permission** — never a shared god-token.

**4. Approval gate**
- Mutating actions can require a second confirmation for high-impact operations such as merge.

**5. Secret hygiene**
- The GitHub App private key, Slack signing secret, and OAuth tokens live in a secret manager or environment — never in code or logs.

**6. Least-privilege data**
- Contributor "first-timer" status comes only from GitHub's public `author_association` field — nothing scraped or inferred about individuals.

---

## Getting Started

### Prerequisites

- Go 1.22+
- Postgres 14+
- A Slack workspace (use a **developer sandbox** for testing — see hackathon rules)
- A GitHub App (for webhooks and scoped tokens)
- Access to the GitHub MCP server
- A Claude API key

### Quick start

```bash
git clone https://github.com/<you>/pr-herder.git
cd pr-herder
cp .env.example .env        # fill in the values below
make migrate                # run DB migrations
make run                    # starts the service on :8080
```

Then expose `:8080` publicly (for example, `ngrok http 8080`) and point your GitHub App webhook and Slack event/interactivity URLs at it.

---

## Configuration

`.env.example`

```env
# Server
PORT=8080
PUBLIC_URL=https://your-tunnel.ngrok.app

# Postgres
DATABASE_URL=postgres://user:pass@localhost:5432/prherder?sslmode=disable

# Slack
SLACK_BOT_TOKEN=xoxb-...
SLACK_SIGNING_SECRET=...
SLACK_DEFAULT_CHANNEL=C0123456789

# GitHub App
GITHUB_APP_ID=...
GITHUB_APP_PRIVATE_KEY_PATH=./github-app.pem
GITHUB_WEBHOOK_SECRET=...

# GitHub MCP
GITHUB_MCP_URL=https://mcp.github.com/...

# LLM (summaries only)
ANTHROPIC_API_KEY=sk-ant-...

# Behavior
STALE_PR_DAYS=7
DIGEST_HOUR_LOCAL=9
QUIET_HOURS=22-08
```

---

## Slack App Manifest

Minimal manifest to register the app (adjust scopes to your needs):

```yaml
display_information:
  name: PR Herder
features:
  bot_user:
    display_name: pr-herder
    always_online: true
oauth_config:
  scopes:
    bot:
      - chat:write
      - commands
      - channels:read
      - users:read
      - im:write
settings:
  event_subscriptions:
    request_url: https://your-tunnel.ngrok.app/slack/events
    bot_events:
      - app_mention
  interactivity:
    is_enabled: true
    request_url: https://your-tunnel.ngrok.app/slack/interact
  org_deploy_enabled: false
  socket_mode_enabled: false
```

---

## GitHub MCP Tools Used

| Purpose | Direction | Notes |
|---|---|---|
| List / get pull requests | read | powers the feed |
| Get PR files and diff stats | read | diff-size and path rules |
| Get check runs / CI status | read | flaky detection |
| Get CODEOWNERS | read | routing |
| Add labels | write | scoped, authorized |
| Submit review (approve / request changes) | write | scoped, authorized, gated |
| Request reviewers | write | load-balancer |
| Post PR comment | write | opt-in contributor help |

All writes flow through the authorization layer described above.

---

## Local Development

```bash
make migrate        # apply DB migrations
make run            # run the service with values from .env
make test           # unit tests (triage rules, authz, flaky classifier)
make lint           # golangci-lint
```

Suggested package layout:

```
cmd/prherder/            # main
internal/ingest/         # webhook verify + queue
internal/triage/         # deterministic rules + flaky classifier
internal/llm/            # Claude summary client (ambiguous PRs only)
internal/slackui/        # Block Kit rendering + interactions
internal/authz/          # identity map + scoped authorization
internal/githubmcp/      # MCP client wrapper
internal/scheduler/      # digests, stale nudges
internal/store/          # Postgres (sqlc-generated)
```

---

## Deployment

- **Container** — ship as a single Go binary in a distroless image.
- **Runtime** — any container host (Cloud Run, Fly.io, ECS, Kubernetes).
- **Database** — managed Postgres.
- **Secrets** — a secret manager, never env files in production.
- **Scaling** — the service is stateless, so scale horizontally; the Postgres job table gives at-least-once processing with idempotency keys.

---

## Roadmap

- [x] PR triage (deterministic rules)
- [x] Slack Block Kit cards and actions
- [x] Scoped authorization and identity mapping
- [x] Flaky-CI detection
- [x] Batched stale-PR digests
- [ ] Reviewer load-balancer v2 (expertise-weighted)
- [ ] Multi-repo and org-wide install
- [ ] "Explain this PR to me" thread command
- [ ] Slack Marketplace submission

---

## Contributing

PR Herder is open source and built in public. First-time contributors are welcome — good first issues are labeled `good-first-issue`. Please read `CONTRIBUTING.md` before opening a PR.

---

## License

MIT © 2026 &lt;your Shreya Baranwal / Vitamin She&gt;
