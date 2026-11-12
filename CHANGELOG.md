# Changelog

All notable changes to Nebula are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

This file documents meaningful release-level changes, not individual commits.

## [0.1.0] — 2026-11-12

First release of **Nebula Orbit**, the platform's EC2-inspired compute service. MVP complete: API, CLI, dashboard, tests and CI.

### Added

- **Orbit MVP** (Phases 0–9): compute service with Docker-backed instances — async provisioning, lifecycle state machine with reconciler, v1 REST API (`orbit/api/openapi.yaml`) + `orbit` CLI, curated images (Ubuntu 24.04, Debian 12) with key-only SSH, SSH key management, private `orbit-net` bridge with port publishing, instance types with capacity/quotas, per-instance data volumes with terminate keep-flag, `orbit ssh` per-platform address resolution.
- **Web dashboard** (`orbit/dashboard`): instance list with state filters and pagination, create wizard with capacity feedback and trust-model acknowledgement, instance detail with lifecycle actions and port editing, SSH key management, toasts, per-section help, light (green) and dark themes.
- **Hardening** (Phase 8): dropped-capability containers, no-new-privileges, private PID/network namespaces, rootfs handling, PIDs/memory/CPU limits, token-authenticated loopback API.
- **Testing & CI**: unit, integration, real-Docker runtime suites, E2E smoke (SSH round-trip, volume persistence, terminate cleanup), dashboard Vitest + Playwright smoke; 4-job GitHub Actions CI.
- **Documentation**: project definition, development guide, architecture overview, Orbit architecture, feature specification, pre-implementation design package, implementation plan (with recorded deviations), end-user usage guide, dashboard UI design system.

### Fixed

- Interactive SSH (PTY) sessions aborted after authentication — instance hardening profile was missing `CAP_AUDIT_WRITE` (required by sshd's login-record path).
- Instance names could not be reused after termination — schema-level UNIQUE replaced with a partial unique index over non-terminated instances (versioned migration v2).
- Dashboard issues: default-key race, 401 handling with re-connect flow, user-data encoding, port-filter regex on the Playwright leg.
