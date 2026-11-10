# Nebula — Architecture Overview

This document describes the system-wide architecture of Nebula: its context, major components, boundaries, and interactions. It deliberately does **not** commit to a final architecture — most of what follows is *proposed* and subject to change. Service-internal design lives in the service architecture documents (currently [orbit.md](orbit.md)).

> **Status key:** items are marked **Proposed** (under consideration, not approved), **Accepted** (agreed direction), or **Implemented** (exists in code). Today, essentially everything is Proposed.

## System Context

Nebula is a self-hostable, cloud-inspired platform. A developer interacts with Nebula to provision and manage cloud-like resources — starting with compute instances — without a real cloud provider.

**Proposed context diagram (textual):**

```text
Developer ──> Nebula (control plane + UI/API) ──> Execution environment (e.g., Docker, expected)
                                                      └── Orbit Instances
```

## High-Level Architecture

The high-level model is **Proposed**:

- **Control plane** — the part of Nebula that receives requests, manages resource state, and coordinates the execution environment. API/UI surface and internal design are undecided.
- **Execution environment** — where resources actually run. For Orbit, this is expected to be Docker emulating compute instances; this expectation is **not yet an approved decision** and should be recorded via an ADR when it is.

No further internal structure (services, layers, storage) has been decided. Do not infer a final architecture from this document.

## Major Component Boundaries

| Component | Responsibility | Status |
|---|---|---|
| Control plane | Request handling, resource state management, orchestration of the execution environment | Proposed |
| Execution environment | Runs Orbit Instances (expected: Docker-based emulation) | Proposed |
| Service modules (e.g., Orbit) | Implement service-specific behaviour on top of the platform model | Proposed |

The interface between the control plane and the execution environment — and whether these are separate processes, libraries, or something else — is undecided.

## Control Plane and Execution Environment Concepts

- The **control plane** owns the authoritative state of resources and exposes the interface users interact with.
- The **execution environment** is subordinate to the control plane: it carries out provisioning and lifecycle operations on actual compute resources.
- This split is a conceptual model for organising the platform, not an assertion about deployment topology.

## Component Interactions

**Proposed flow for Orbit instance provisioning (illustrative, not final):**

1. User requests instance creation through the Nebula interface.
2. Control plane validates the request, allocates resources, and records instance state.
3. Control plane instructs the execution environment to create the emulated instance.
4. Execution environment confirms; control plane updates instance state.

Sequence details, protocols, and state stores are undecided.

## Data Flows

- All resource state is expected to flow through the control plane; the execution environment reports the outcome of operations. Where state is persisted and how it is structured are **undecided** and must not be invented here.

## Service-Level Architecture

- **Orbit (compute service):** [orbit.md](orbit.md)

New services get their own architecture document under `docs/architecture/` only when work on them begins.
