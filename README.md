# Nebula

[![CI](https://github.com/krishnasinghcode/nebula/actions/workflows/ci.yml/badge.svg)](https://github.com/krishnasinghcode/nebula/actions/workflows/ci.yml)

**Nebula** is an open-source, cloud-inspired development platform: it provisions and runs
commonly-used cloud infrastructure on your own machine, without cloud-provider costs.

> **Status: v0.1.0 — Orbit MVP released.** Instance provisioning, SSH access, lifecycle,
> volumes, capacity quotas, REST API, CLI and web dashboard — implemented, tested and
> CI-gated. Development continues on the roadmap in the implementation plan.

## Why Nebula?

Developers who want to learn, prototype, or build against cloud-style infrastructure are
forced to use real cloud providers — incurring cost, complexity, and account friction.
Nebula provides a practical, self-hostable alternative: cloud-like services provisioned
through cloud-familiar interfaces, at little to no cost.

Development is incremental: practical versions of the most commonly-used services, not an
AWS replica.

## Nebula Orbit — compute service

**Orbit** is Nebula's compute service, inspired by EC2. Create and manage **Orbit
Instances** — cloud-like compute nodes — with real SSH access, published ports, volumes,
capacity quotas and a web dashboard.

The initial runtime emulates instances with hardened containers (ADR-0001): dropped
capabilities, no-new-privileges, private PID/network namespaces, per-instance SSH.

**Implemented in v0.1.0:**

- Instance provisioning — curated images (Ubuntu 24.04, Debian 12), instance types, EC2-style user-data
- Lifecycle — start / stop / reboot / terminate with a proper state machine
- SSH access — key-only auth; private keys never reach the server
- Private networking and published ports (SSH/22 auto-published)
- Data volumes that survive restarts and terminate/recreate
- Capacity admission control and quotas (vCPU, memory, instance count)
- Token-authenticated REST API with an OpenAPI contract
- CLI (`orbit`) and web dashboard (light & dark themes, state filters, toasts)

## Limitations — read before using

Orbit is a **local development and learning tool**, not a production cloud:

- **Everything runs on your host.** Instances share your machine's CPU, memory, disk and
  network. Quotas only partition *your* resources — there is no more capacity than your
  PC has. Heavy instance workloads hit *your* limits.
- **Instances are containers, not VMs.** They share the host kernel and are **not**
  security isolation. A root user inside an instance can affect your host. Only run
  workloads and user data you trust — Orbit asks you to acknowledge this on every create.
- **Simulation-level fidelity.** Cloud-familiar concepts and workflows, deliberately
  simplified: single host only (no multi-node scheduling), no IAM/roles, no snapshots or
  AMIs, monitoring is partial, the API surface is a small subset of EC2.
- **Local only.** No public networking, no high availability, no managed services. The
  dashboard and API bind to loopback by default.
- **Docker required.** Orbit depends on a running Docker Engine (its compute runtime).

## Quickstart

**Full end-to-end walkthrough (backend, dashboard, token, keys, create, SSH, terminate):**
[docs/guides/orbit-usage.md](docs/guides/orbit-usage.md)

Prerequisites: Docker running, Go 1.27. Build: `cd orbit && go build ./cmd/orbit`.

```sh
./orbit serve                          # control plane (API on 127.0.0.1:7420)
./orbit key create dev                 # generate + register an SSH key
./orbit images                         # curated image catalogue
./orbit instance create web-1 \
  --image ubuntu-24.04 --type orbit.micro --key dev \
  --port 8080:80                       # 8080 (host) -> 80 (instance), loopback
./orbit instance list                  # wait for state: running
./orbit ssh web-1                      # SSH in (key handled automatically)
./orbit instance terminate web-1
```

Want the web dashboard instead of the CLI? See [docs/development.md](docs/development.md)
(§ Dashboard) — `orbit/dashboard`, Vite + React + TypeScript.

## Documentation Index

| Document | What it's for |
|---|---|
| **Getting started** | |
| [docs/guides/orbit-usage.md](docs/guides/orbit-usage.md) | End-user walkthrough: run Orbit end-to-end, use the dashboard, SSH into instances |
| [docs/development.md](docs/development.md) | Prerequisites, build, test commands, dashboard dev setup — for contributors |
| **Project** | |
| [docs/project.md](docs/project.md) | Authoritative scope: vision, goals, non-goals (what Nebula will/won't be) |
| [AGENTS.md](AGENTS.md) | Working rules for all contributors (human or AI): git workflow, doc policies |
| **Architecture** | |
| [docs/architecture/overview.md](docs/architecture/overview.md) | System-wide components, boundaries, status key (Proposed/Accepted/Implemented) |
| [docs/architecture/orbit.md](docs/architecture/orbit.md) | Orbit's internal architecture: control plane, runtime, reconciler, networking |
| **Orbit feature specs** | |
| [docs/features/orbit/instance-management.md](docs/features/orbit/instance-management.md) | The instance-management feature specification (requirements & acceptance criteria) |
| [docs/features/orbit/design/README.md](docs/features/orbit/design/README.md) | Pre-implementation design package index: requirements, priorities, architecture, security, data/API, testing |
| [docs/features/orbit/design/dashboard-design.md](docs/features/orbit/design/dashboard-design.md) | Dashboard UI design system: themes, tokens, patterns (keep new UI consistent) |
| [docs/features/orbit/design/implementation-plan.md](docs/features/orbit/design/implementation-plan.md) | Implementation record: phases, deviations from spec, open questions |
| **Decisions** | |
| [docs/decisions/0001-orbit-technology-stack.md](docs/decisions/0001-orbit-technology-stack.md) | ADR: Go + Chi + Docker + SQLite + React stack, with alternatives considered |
| [CHANGELOG.md](CHANGELOG.md) | Meaningful changes per release |

Rule of thumb: *how do I use it* → guide · *how do I build it* → development.md ·
*what is it / won't it be* → project.md · *how does it work* → architecture ·
*why was X decided* → ADRs and the implementation plan.

## Contributing

Contributions are welcome:

1. Read [AGENTS.md](AGENTS.md) — working rules and documentation policies for all contributors.
2. Read [docs/project.md](docs/project.md) for scope and non-goals.
3. Check open Issues, or open one using the Issue templates before starting significant work.
4. Follow the Git and PR conventions in [AGENTS.md](AGENTS.md); PRs require human review.
