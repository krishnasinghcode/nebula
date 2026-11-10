# Changelog

All notable changes to Nebula are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

This file documents meaningful release-level changes, not individual commits.

## [Unreleased]

### Added

- **Orbit MVP** (Phases 0–7): compute service with Docker-backed instances — async provisioning, lifecycle state machine with reconciler, v1 REST API (`orbit/api/openapi.yaml`) + `orbit` CLI, curated images (Ubuntu 24.04, Debian 12) with key-only SSH, SSH key management, private `orbit-net` bridge with port publishing, instance types with capacity/quotas, per-instance data volumes with terminate keep-flag, `orbit ssh` per-platform address resolution.
- Documentation foundation: project definition, development guide, architecture overview, Orbit architecture, Orbit instance management feature specification, GitHub Issue and PR templates, and agent instructions (`AGENTS.md`).
