# Nebula

Nebula is an open-source-powered, cloud-inspired development platform that lets developers provision and interact with commonly used cloud infrastructure without depending on expensive cloud providers.

> **Status:** The project is in early development. Documentation is being established before application code exists. Everything described here as *planned* is not yet implemented.

## Why Nebula?

Developers who want to learn, prototype, or build against cloud-style infrastructure are currently forced to use real cloud providers — incurring cost, complexity, and account setup friction. Nebula aims to provide a practical, self-hostable alternative: a platform where commonly used cloud services can be provisioned and managed through a cloud-like interface, at little to no cost.

Nebula is developed incrementally. Rather than replicating all AWS functionality, it implements practical versions of the most commonly used services.

## Current Capabilities

- **None yet.** Nebula is pre-implementation. See [docs/project.md](docs/project.md) for scope and direction.

## Nebula Orbit (first service)

**Orbit** is Nebula's compute service, inspired by EC2. Its initial objective is to let users create and manage compute instances — **Orbit Instances** — through a cloud-like interface.

Planned capabilities (not yet implemented):

- Instance provisioning
- Instance lifecycle management
- Resource allocation
- SSH access
- Instance monitoring and management

The initial implementation is expected to use Docker to emulate compute instances; this approach has not been finalised.

See [docs/architecture/orbit.md](docs/architecture/orbit.md) and [docs/features/orbit/instance-management.md](docs/features/orbit/instance-management.md).

## Architecture

System-level architecture, component boundaries, and design direction are documented in [docs/architecture/overview.md](docs/architecture/overview.md).

## Setup and Usage

Setup instructions, prerequisites, and development commands are maintained in [docs/development.md](docs/development.md). These will be filled in as the technology stack is selected.

## Contributing

Contributions are welcome. To contribute:

1. Read [AGENTS.md](AGENTS.md) — it defines the working rules and documentation policies that apply to all contributors (human or AI).
2. Read [docs/project.md](docs/project.md) for scope and non-goals.
3. Check open Issues, or open one using the Issue templates before starting significant work.
4. Follow the Git and PR conventions in [AGENTS.md](AGENTS.md); pull requests require human review before merging.
