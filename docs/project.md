# Nebula — Project Definition

This document is the authoritative definition of what Nebula is, who it serves, and what it will and will not do. When any other document conflicts with this one on scope or direction, this document wins.

## Vision

A platform where any developer can provision and interact with cloud-like infrastructure — instances, and over time other commonly used services — on their own machine or a cheap host, without cloud provider costs or account complexity.

## Problem Statement

Cloud providers are the default way to learn, prototype, and build against infrastructure such as virtual machines. But they are expensive, require accounts, billing setup, and quotas, and are often excessive for learning, experimentation, and small projects. Developers therefore face a barrier between themselves and cloud-style infrastructure experience.

Nebula addresses this by providing a practical, open-source, self-hostable platform that emulates the commonly used parts of cloud services locally or on modest hardware.

## Target Users

- **Developers learning cloud concepts** who want hands-on experience without cloud costs.
- **Developers prototyping or experimenting** who need ephemeral cloud-like resources.
- **Small teams and homelab users** who want self-hosted infrastructure management.

## Core Use Cases

1. Provision and manage compute instances (Orbit Instances) through a cloud-like interface.
2. Access instances (e.g., via SSH) as if they were real VMs.
3. Practice cloud workflows — provisioning, lifecycle management, monitoring — locally.

Use cases beyond compute will be added as new services are delivered.

## Product Principles

1. **Open source** — developed in the open, community-oriented.
2. **Practical over complete** — ship practical versions of common services; never try to replicate all AWS functionality.
3. **Cloud-familiar** — model concepts and workflows on what developers already know.
4. **Incremental** — one service at a time; small, working increments over grand designs.
5. **Honest** — always distinguish planned, proposed, and implemented functionality.

## Scope

**Current scope (implemented):** none — the project is pre-implementation.

**Current scope (in progress/planned):**

- **Nebula Orbit** — EC2-inspired compute service. Planned capabilities: instance provisioning, instance lifecycle management, resource allocation, SSH access, instance monitoring and management. The initial implementation is expected to use Docker to emulate compute instances (not yet approved as a final decision).

## Non-Goals

- Replicating the full feature set of AWS or any specific cloud provider.
- Providing production-grade parity with real cloud services (SLAs, multi-region, compliance certifications).
- Competing on scale, availability guarantees, or enterprise support.
- Supporting arbitrary cloud services beyond the commonly used ones selected for incremental delivery.

## Long-Term Direction

Orbit is the first service. Over time, Nebula is expected to grow additional services — practical versions of other commonly used cloud infrastructure — sharing a consistent platform model (concepts such as a control plane and execution environment). Specific future services are intentionally not committed to here; they will be defined in their own architecture and feature documents when work begins.
