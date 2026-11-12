# Nebula — Development Guide

This document covers prerequisites, setup, build, and testing for working on Nebula.

## Prerequisites

- Git
- Go 1.27 (backend and CLI)
- Node.js 20+ (dashboard development only)
- Docker — required: it is Orbit's compute runtime (ADR-0001)

## Technology Stack

Accepted — see [ADR-0001](decisions/0001-orbit-technology-stack.md) for rationale and alternatives considered.

| Technology | Purpose |
|---|---|
| **Go** | Backend and infrastructure orchestration (control plane + runtime integration). |
| **Chi** | HTTP routing for the REST API. |
| **Docker (`github.com/docker/docker` client)** | Compute runtime integration (minimum Engine API version pinned in CI). |
| **SQLite (`modernc.org/sqlite`)** | Embedded state store; no database server install required. |
| **Cobra** | CLI framework. |
| **OpenAPI** | API contract; TypeScript client generated from the spec. |
| **Polling (2s)** | Instance/key status updates in the dashboard. *(ADR-0001 listed SSE; implementation uses polling — logged deviation in the implementation plan.)* |
| **React + TypeScript + Vite + Tailwind CSS (plain, no shadcn/ui)** | Management dashboard. *(shadcn/ui was listed in ADR-0001; plain Tailwind was chosen — logged deviation in the implementation plan.)* |

Distributed infrastructure (Redis, Kafka, etc.) is intentionally excluded.

## Environment Configuration

- Orbit control plane stores state and its API token under `~/.orbit/` (token file `0600`, generated at first `orbit serve`).
- Tests on Windows may need `GOTMPDIR` set to a writable, policy-approved temp dir if the default is blocked.
- `.gitignore` at repo root covers Go binaries, `node_modules/`, and the dashboard build output.

## Setup Instructions

```sh
git clone <repo-url> && cd nebula
cd orbit
go build ./...
```

## Build and Execution

```sh
# run control plane (localhost only, default port 7420)
go run ./cmd/orbit serve
# CLI
go run ./cmd/orbit version
```

## Testing

```sh
go test ./...            # unit + integration
go test ./... -tags=runtime   # real-Docker suite (Phase 1+) — requires Docker running
```

- All new functionality must ship with appropriate tests (see `AGENTS.md`, section G).
- Bug fixes must include regression tests.
- Runtime-tagged tests require a running Docker Engine.

## Dashboard (Phase 9)

A web UI lives in `orbit/dashboard/` (React + TypeScript + Vite + Tailwind). The API client is generated from `orbit/api/openapi.yaml` (`npm run generate:api`).

```sh
orbit serve                 # terminal 1 — control plane on 127.0.0.1:7420
cd orbit/dashboard
npm install
npm run generate:api
npm run dev                 # terminal 2 — UI on http://127.0.0.1:5173
```

Paste the API token (`~/.orbit/token`) when prompted by the UI. Tests: `npm test` (components), `npx playwright test` (smoke; needs a running `orbit serve` and `ORBIT_TOKEN`).

## Development Conventions

- Follow the working rules in [`AGENTS.md`](../AGENTS.md) — especially the documentation update matrix and testing requirements.
- Use the terminology defined in [`AGENTS.md`](../AGENTS.md) (Nebula, Orbit, Orbit Instance) consistently in code, tests, and documentation.
- Keep changes small and focused; link every PR to an Issue.
