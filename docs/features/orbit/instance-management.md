# Feature Specification: Orbit Instance Management

Service: **Orbit** · Status: **In development — MVP implemented behind the API/CLI; release pending security gates (Phase 8)**

This document specifies the Orbit Instance Management feature. Placeholders marked `<TBD>` indicate requirements or decisions that have not been established; they must not be filled in with invented details.

> A detailed pre-implementation design package for this feature (requirements, priorities, architecture, technical design, security, data/API, testing) lives in [design/README.md](design/README.md). Where this spec and the design package conflict, the design package — once approved — takes precedence and this spec should be updated.

## 1. Objective

Enable users to create and manage compute instances (**Orbit Instances**) through a cloud-like interface: provisioning, lifecycle management, resource allocation, SSH access, and monitoring.

## 2. User Workflow

**Proposed** (final UX undecided):

1. User opens the Nebula interface for Orbit.
2. User requests a new Orbit Instance, specifying `<TBD>` name, and resource requirements (CPU, memory) `<TBD>`.
3. Orbit provisions the instance; the user sees it transition Provisioning → Running.
4. User manages the instance: start, stop, terminate `<TBD>`.
5. User connects to a running instance over SSH `<TBD>`.
6. User views instance status and basic monitoring `<TBD>`.

## 3. Functional Requirements

| # | Requirement | Status |
|---|---|---|
| FR-1 | Users can provision an Orbit Instance | Implemented (API + CLI; async provisioning) |
| FR-2 | Users can view their instances and their current state | Implemented (list + describe with events) |
| FR-3 | Users can start and stop instances | Implemented |
| FR-4 | Users can terminate instances (terminal) | Implemented (optional keep-root-volume) |
| FR-5 | Users can specify resource requirements (CPU, memory) at provisioning | Implemented (instance-type catalogue; admission control + quotas) |
| FR-6 | Users can access running instances over SSH | Implemented (real sshd in guest; key-only auth; `orbit ssh`) |
| FR-7 | Users can view instance status/monitoring information | Implemented (state, events, capacity; logs/metrics deferred) |

Additional requirements (authentication, authorisation, multi-user support): `<TBD>`.

## 4. Technical Design

**TBD.** Expected direction (not approved): the initial implementation uses Docker to emulate compute instances. Internal design — component breakdown, runtime abstraction, state handling — is documented as open questions in [orbit.md](../../architecture/orbit.md). No design decisions have been accepted.

## 5. API Contracts

**TBD.** No API exists yet; endpoints, protocols, and formats will be defined here once designed. When APIs are adopted, an OpenAPI specification will be introduced (see [AGENTS.md](../../../AGENTS.md) documentation update matrix). **Do not invent endpoints before they are specified here.**

## 6. Data Model

**TBD.** Instance metadata to persist (e.g., identifier, state, allocated resources) and its storage have not been designed. No schema exists; none should be assumed.

## 7. Edge Cases

To be addressed in design (list is illustrative, not exhaustive):

- Provisioning failure partway through (partial resources allocated).
- Operations on instances in non-compatible states (e.g., stopping a stopped instance).
- Resource exhaustion — cannot allocate requested CPU/memory.
- Terminating an instance that is currently provisioning or running.
- Instance unreachable/hung during lifecycle operations.
- SSH access attempt on a stopped instance.

## 8. Security Considerations

- **Isolation:** instances must be isolated from each other and from the host; with Docker-based emulation this is weaker than true VM isolation and must be documented honestly.
- **SSH credentials:** key generation, storage, and authentication model `<TBD>`.
- **Access control:** who may create/manage instances `<TBD>`.
- **Resource limits:** quotas to prevent abuse of host resources `<TBD>`.
- **Input validation:** all provisioning parameters validated at the boundary.

## 9. Acceptance Criteria

Acceptance criteria will be defined per GitHub Issue when implementation begins. Indicative checks for this feature:

- [ ] An Orbit Instance can be provisioned and reaches the Running state.
- [ ] Instance state is observable by the user.
- [ ] Start/stop/terminate operations work and enforce valid state transitions.
- [ ] Resource requirements are applied to instances.
- [ ] SSH access to a running instance works.
- [ ] Monitoring/status information is available for instances.
- [ ] Appropriate tests cover success and failure paths.

## 10. Known Limitations

- Local test execution on the primary development machine is blocked by Windows Device Guard; the test suites run in CI. See the implementation plan's deviations log.
- Volume quotas are estimate-based (Docker local volumes have no native size enforcement).
- Instance-to-instance traffic on the shared `orbit-net` bridge is unrestricted by design (single-user local trust model).
- Rootfs is not read-only yet; hardening polish is tracked for Phase 8.
- Docker-based emulation means instances are not true VMs (isolation and behaviour differences).
- Single-host operation assumed initially; distributed execution is out of scope `<TBD>`.
