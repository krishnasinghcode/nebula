# 0001. Orbit Technology Stack

## Status

Accepted (2026-11-10)

## Context

Nebula Orbit needs a technology stack selected before implementation begins. Constraints from the project definition and design package (`docs/features/orbit/design/`): single-host, laptop-friendly, local-first tool; modular monolith control plane behind a runtime adapter interface; Docker as the initial compute runtime; CLI-first MVP with a web dashboard later; portable across Linux, macOS, and Windows (Docker Desktop); minimal dependencies; no distributed infrastructure.

## Options Considered

- **Backend language:** Go (official Docker SDK, single static binary, strong concurrency for the reconciler) vs. Rust (smaller ecosystem for Docker SDK, slower iteration) vs. Python/Node (weaker fit for system-level orchestration, distribution friction).
- **HTTP framework:** Chi (stdlib-compatible, minimal) vs. Gin/Echo/Fiber (heavier conventions, less stdlib alignment).
- **State store:** SQLite (embedded, zero-ops, WAL) vs. PostgreSQL (requires a server install — wrong for a local-first tool) vs. flat files (no transactions/concurrency).
- **SQLite driver:** `modernc.org/sqlite` (pure Go, portable cross-compilation) vs. `mattn/go-sqlite3` (CGO complicates multi-OS single-binary builds).
- **CLI:** Cobra (de-facto Go standard, subcommand/flags/help ergonomics) vs. stdlib flag (insufficient for a multi-command cloud-style CLI).
- **Frontend:** React + TypeScript + Vite + Tailwind + shadcn/ui (mature, typed, component ecosystem) vs. no dashboard in MVP (CLI-only). Decision: adopt the stack now, but sequence dashboard work after the MVP API is stable.
- **Real-time updates:** Server-Sent Events (one-way, lightweight, HTTP-native) vs. WebSockets (two-way, more complexity than needed for status streaming).

## Decision

Adopt the following stack for Orbit:

| Technology | Purpose |
|---|---|
| **Go** | Backend and infrastructure orchestration (control plane, execution plane integration). |
| **Chi** | HTTP routing for the REST API. |
| **`github.com/docker/docker` client** | Docker Engine API integration (pinned minimum Engine API version). |
| **SQLite via `modernc.org/sqlite`** | Embedded metadata/state store (pure Go, no CGO, portable single binary). |
| **Cobra** | CLI framework. |
| **OpenAPI** | API contract; TypeScript client/types generated from the spec for the frontend. |
| **SSE** | Streaming instance status, lifecycle events, and monitoring updates. |
| **React + TypeScript + Vite + Tailwind CSS + shadcn/ui** | Management dashboard (built after the MVP API is stable; CLI remains the primary interface). |

Control plane and execution-plane integration share one Go codebase with strict package boundaries; only the runtime adapter layer imports the Docker client. Docker is the initial runtime; the adapter interface preserves the path to QEMU/KVM. Redis, Kafka, and other distributed infrastructure are intentionally excluded.

## Consequences

- Single portable binary for control plane + CLI across all three OSes (no CGO).
- Frontend integrates against a machine-readable contract (OpenAPI → generated TS client), keeping the spec authoritative.
- Curated Orbit images (Dockerfiles + CI build) are part of the stack and must be maintained.
- Docker Desktop behaviour differences (capacity envelope, port publishing for SSH) apply as documented in the design package.
- Dashboard development must not drive API changes until the MVP API is frozen (Phase 2+ of the implementation plan).
