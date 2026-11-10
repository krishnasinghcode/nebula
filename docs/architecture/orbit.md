# Orbit — Architecture

This document describes the internal architecture of the **Orbit** compute service. Orbit is **planned, not implemented**. Everything below reflects the currently proposed direction; undecided items are explicitly marked **TBD** and must not be filled in with invented details.

> **Status key:** **Proposed** (under consideration), **Accepted** (agreed), **Implemented** (exists in code). Orbit's overall concept is Proposed; its internal design is largely TBD.

For the platform-wide context, see [overview.md](overview.md). For the functional specification, see [instance-management.md](../features/orbit/instance-management.md).

## Responsibilities

Orbit is Nebula's compute service, inspired by EC2. It is responsible for:

- Provisioning Orbit Instances on request.
- Managing Orbit Instance lifecycle (create, start, stop, terminate).
- Allocating resources (CPU, memory) to instances.
- Providing access to instances (expected: SSH).
- Monitoring and reporting instance status.

## Internal Components

**Approved direction (ADR-0001):** Orbit is a modular monolith in Go — Chi-served REST API, instance manager, scheduler/capacity manager, network/storage/key managers, reconciler, and a SQLite state store (`modernc.org/sqlite`), with a Docker runtime adapter as the only Docker-importing package. Package boundaries enforce the control-plane/execution-plane split; full responsibilities are in [design/architecture.md](../features/orbit/design/architecture.md). Component-level details (exact package layout) are finalised in Phase 0–2 of the [implementation plan](../features/orbit/design/implementation-plan.md).

## Instance Lifecycle

**Proposed** lifecycle states (drawn from cloud-familiar models; not yet finalised):

```text
Provisioning ──> Running ──> Stopped ──> (Running | Terminated)
      │
      └──────────────────────> Terminated
```

- **Provisioning** — resources are being allocated and the instance created.
- **Running** — the instance is available and accessible.
- **Stopped** — the instance exists but is not consuming compute.
- **Terminated** — the instance is deleted; a terminal state.

Transitions, allowed operations per state, and failure handling are **TBD**.

## Resource Allocation

**TBD.** Users are expected to specify resource requirements (CPU, memory) when provisioning an instance. How resources map onto the execution environment (e.g., Docker resource limits, if that approach is adopted), and whether allocation is enforced or advisory, is undecided.

## Runtime Integration

**Accepted (ADR-0001):** the initial implementation uses **Docker** to emulate compute instances — each Orbit Instance backed by a container-like unit, integrated via the official Docker Go client behind the runtime adapter (which preserves the option of a future QEMU/KVM backend; see [design/architecture.md §Evolution](../features/orbit/design/architecture.md)). The adapter's create path enforces the security profile in [design/security.md](../features/orbit/design/security.md).

## Networking and SSH

- **SSH access** is a planned capability: users should be able to connect to a running Orbit Instance over SSH.
- **TBD:** how SSH is exposed (port mapping, key management, authentication model), and how instance networking is modelled and isolated. No network design exists yet.

## Persistence and Metadata

**Accepted (ADR-0001):** instance metadata is persisted in SQLite (WAL mode) — the state store is authoritative; the runtime is only observed via the reconciler. The concrete schema is defined in [design/data-and-api.md](../features/orbit/design/data-and-api.md) and implemented in Phase 2 of the plan.

## Error Handling

**TBD.** Expected principles (inherited from [AGENTS.md](../../AGENTS.md) section C): fail explicitly, surface clear errors to users, and keep instance state consistent on failure. Specific error taxonomy and recovery behaviour are undecided.

## Security and Isolation

Security-sensitive areas, all **TBD** at the design level:

- **Isolation between Orbit Instances** — how strongly instances are isolated from each other and from the host (this is a key difference from real VMs if Docker emulation is used; it must be documented honestly).
- **SSH credentials and key management.**
- **API access control** — who can create/manage instances (multi-user vs single-user is undecided).
- **Resource abuse prevention** — quotas or limits on instance count/resources.

## Component Interactions

**Proposed (illustrative) interaction for provisioning:**

1. Orbit receives a create-instance request (via the Nebula control plane / API surface).
2. Orbit validates the request and allocates resources.
3. Orbit instructs the execution environment to create the instance unit.
4. Orbit records instance metadata and returns the instance (state: Provisioning → Running).

Component boundaries, protocols, and state stores are TBD.
