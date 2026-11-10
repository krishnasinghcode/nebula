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

`<TBD>` — environment variables, configuration files, and secrets handling will be documented here as the configuration approach is implemented (Phase 0–2 of the [implementation plan](features/orbit/design/implementation-plan.md)).

## Setup Instructions

`<TBD>` — clone and bootstrap steps to be added after the stack is selected.

## Build and Execution

`<TBD>` — commands to build and run the platform locally will be added here.

## Testing

`<TBD>` — commands to run unit and integration tests will be added here. In the meantime:

- All new functionality must ship with appropriate tests (see `AGENTS.md`, section G).
- Bug fixes must include regression tests.

## Development Conventions

- Follow the working rules in [`AGENTS.md`](../AGENTS.md) — especially the documentation update matrix and testing requirements.
- Use the terminology defined in [`AGENTS.md`](../AGENTS.md) (Nebula, Orbit, Orbit Instance) consistently in code, tests, and documentation.
- Keep changes small and focused; link every PR to an Issue.
