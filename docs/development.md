# Nebula — Development Guide

This document covers prerequisites, setup, build, and testing for working on Nebula. It is a template until the technology stack is selected; placeholders marked `<TBD>` must be filled in as decisions are made.

## Prerequisites

- Git
- `<TBD>` — runtime/toolchain requirements once the stack is selected
- Docker — expected to be required for the initial Orbit implementation (not yet approved; treat as provisional)

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
| **SSE** | Real-time instance status/lifecycle/monitoring updates. |
| **React + TypeScript + Vite + Tailwind CSS + shadcn/ui** | Management dashboard (built after the MVP API is stable; CLI-first until then). |

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

## Development Conventions

- Follow the working rules in [`AGENTS.md`](../AGENTS.md) — especially the documentation update matrix and testing requirements.
- Use the terminology defined in [`AGENTS.md`](../AGENTS.md) (Nebula, Orbit, Orbit Instance) consistently in code, tests, and documentation.
- Keep changes small and focused; link every PR to an Issue.
