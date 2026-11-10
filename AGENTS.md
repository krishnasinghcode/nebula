# AGENTS.md — Instructions for AI Agents

This file defines the working rules for every AI coding agent operating in this repository, regardless of tool (OpenCode, Claude Code, Copilot, etc.). These rules take precedence over general defaults.

---

## A. Project Context and Documentation Hierarchy

### Project Description

**Nebula** is an open-source-powered, cloud-inspired development platform that lets developers provision and interact with commonly used cloud infrastructure without depending on expensive cloud providers. It is developed incrementally, delivering practical versions of commonly used cloud services rather than replicating all AWS functionality.

**Orbit** is Nebula's first service: an EC2-inspired compute service. Its initial objective is to let users create and manage compute instances ("Orbit Instances") through a cloud-like interface. The initial implementation is expected to use Docker to emulate compute instances, but no architectural decisions beyond this have been approved.

### Product Objectives and Principles

1. **Accessibility** — provide cloud-like infrastructure experience without cloud provider costs.
2. **Incremental delivery** — implement practical versions of services one at a time; do not over-architect.
3. **Cloud-familiar model** — use concepts and workflows developers already know.
4. **Honest documentation** — clearly distinguish **planned**, **proposed**, and **implemented** functionality.
5. **Minimal, scalable structure** — documentation and code grow only as needed.

### Terminology

| Term | Meaning |
|---|---|
| **Nebula** | The platform. |
| **Orbit** | Nebula's compute service. |
| **Orbit Instance** | A compute resource managed by Orbit. |
| **Control plane** | The part of Nebula that manages resources (proposed concept, not yet designed). |
| **Execution environment** | Where Orbit Instances run (expected: Docker emulated; not yet approved). |

Never use "EC2", "container", or "VM" interchangeably with Orbit Instance when describing Nebula concepts.

### Documentation Files and Their Purpose

| File | Purpose |
|---|---|
| `AGENTS.md` | This file. Working rules for all agents. |
| `README.md` | Public-facing introduction, setup, and contribution info. |
| `CHANGELOG.md` | Meaningful release changes (version-based sections). |
| `docs/project.md` | Authoritative project definition: vision, scope, non-goals, principles. |
| `docs/development.md` | Prerequisites, setup, build, testing instructions. |
| `docs/architecture/overview.md` | System-wide architecture: components, boundaries, interactions. |
| `docs/architecture/orbit.md` | Orbit's internal architecture and design. |
| `docs/features/orbit/instance-management.md` | Orbit Instance feature specification. |
| `docs/decisions/` | Architecture Decision Records (ADRs); created when a significant decision is made. |
| `.github/` | Issue and pull request templates. |

### Where Information Belongs

- **What the product is and is not** → `docs/project.md`
- **How to set up, build, test** → `docs/development.md`
- **How the whole system fits together** → `docs/architecture/overview.md`
- **How a specific service works internally** → `docs/architecture/<service>.md`
- **What a feature must do** → `docs/features/<service>/<feature>.md`
- **Why a significant decision was made** → `docs/decisions/` (ADR)
- **What changed in a release** → `CHANGELOG.md`
- **How to contribute** → `README.md`

Do not duplicate information across files. Link instead.

---

## B. Mandatory Reading

Before starting any task, agents must read, in order:

1. **This file (`AGENTS.md`)** — always.
2. **The relevant feature specification** under `docs/features/` — when the task touches a feature.
3. **Relevant architecture documentation** — when the task touches components described there.
4. **Related implementation and tests** — inspect code and tests adjacent to the change.
5. **The associated GitHub Issue and its acceptance criteria** — when available.
6. **`docs/development.md`** — when setup, tooling, or testing instructions are relevant.

Reading all documents for every small change is not required. Use judgement: read only what the task affects.

---

## C. Development Guidelines

These are technology-agnostic conventions. The technology stack is not yet selected; do not add or enforce stack-specific rules here.

- **Code organisation and modularity** — keep components small and focused; separate concerns clearly; group code by service and feature; avoid circular dependencies.
- **Naming and formatting** — use clear, descriptive names consistent with Nebula/Orbit terminology; follow a single formatting convention per file type; format code automatically where tooling allows.
- **Error handling and logging** — fail explicitly rather than silently; never swallow errors without handling them; log with appropriate severity and enough context to debug; never log secrets.
- **Input validation** — validate all external input (user input, API requests, configuration) at the boundary; reject invalid input with clear errors.
- **API design** — design APIs to be cloud-like and consistent across services; keep contracts explicit and documented in feature specifications; do not invent API endpoints before they are specified.
- **Security** — apply least privilege; never hardcode credentials; treat instance access (e.g., SSH) as a security-sensitive surface; validate and sanitise everything from users.
- **Testing** — write tests for every meaningful behaviour; prefer fast, isolated tests; tests must cover both success and failure paths.
- **Dependency management** — add dependencies sparingly; justify each new dependency; keep them pinned/updated and free of known vulnerabilities.
- **Maintainability and performance** — favour readable solutions over clever ones; measure before optimising; keep resource usage proportional to the problem.

---

## D. Git and GitHub Workflow

### Branches
- Use feature branches: `feat/<short-name>`, `fix/<short-name>`, `docs/<short-name>`, `chore/<short-name>`.
- Branch from the default branch; keep branches focused on one task.

### Commits
- Write meaningful, imperative commit messages (e.g., `Add orbit instance start action`).
- Keep commits atomic; do not mix unrelated changes.

### Issues
- Work from GitHub Issues when available; each task should map to an Issue with clear acceptance criteria.
- Use sub-issues for large work items broken into parts.
- Use GitHub Projects to track status.

### Priority and status updates
- Keep Issue status current while working (e.g., In Progress → In Review → Done).
- Add a comment to the Issue when status materially changes or blockers are found.

### Pull requests
- Open a PR for every change, even small ones.
- Link the PR to its Issue ("Closes #N" or a plain reference).
- Keep PRs reviewable in size; describe what and why.

### Implementation reports
Every completed task must produce a report (see section G).

### Authorisation and approvals
Agents **may**, when authorised: create branches, commit, push, create Issues, update Projects, open PRs.

Agents **require human approval before**: merging PRs, publishing releases, deleting important resources, or changing repository permissions (also see section H).

---

## E. Documentation Update Policy

Mandatory documentation review matrix:

| Change type | Documentation to review/update |
|---|---|
| New feature | Relevant feature specification |
| Feature modification | Relevant feature specification |
| Bug fix | Tests; feature documentation only if documented behaviour changes |
| Refactoring | Architecture only if component responsibilities or boundaries change |
| API change | Relevant feature specification; introduce/update OpenAPI when adopted |
| Database schema change | Migration and relevant feature specification |
| Architecture change | Relevant architecture documents and ADR if significant |
| New service | Project/architecture overview and service-specific documentation |
| Technology/dependency change | Development and architecture documentation; ADR if significant |
| Environment/setup change | Development documentation |
| Security change | Relevant architecture or security documentation |
| Project scope change | Project definition |
| Release | CHANGELOG |

Additional rules:

- Review documentation impact for **every** task.
- Update only documents that have become inaccurate or incomplete.
- Do not rewrite unrelated documents.
- Never duplicate information across documents — link instead.
- Never document planned functionality as implemented.
- Do not update `CHANGELOG.md` for every commit — only for meaningful release changes.
- Keep documentation consistent with the actual codebase.
- A documentation review does not automatically require an edit.

---

## F. Architecture Documentation Policy

- Maintain **one** system-wide architecture overview (`docs/architecture/overview.md`).
- Maintain a **separate** architecture document per service (`docs/architecture/orbit.md` for Orbit).
- The overview explains how the platform's major components interact; the service document explains internal components and implementation.
- Feature specifications describe expected functionality, not the entire system architecture.
- Create ADRs in `docs/decisions/` only for significant architectural decisions. ADR structure: 1. Title, 2. Status, 3. Context, 4. Options considered, 5. Decision, 6. Consequences.
- Clearly distinguish **proposed**, **accepted**, and **implemented** designs in architecture documents.
- Do not create hypothetical architecture documents for future services.

---

## G. Testing and Completion Requirements

Code is not complete merely because implementation has finished. Before declaring a task complete:

- Write appropriate unit and integration tests.
- Add regression tests for bug fixes.
- Verify the build and tests pass.
- Review documentation per the matrix in section E.
- Keep GitHub Issue/Project status accurate.

### Completion Report

Every completed task must produce a report containing:

1. Implementation summary.
2. Files created or modified.
3. Tests executed and results.
4. Documentation updated.
5. Documentation reviewed but unchanged.
6. Architecture impact.
7. Known limitations.
8. GitHub Issue and PR references.

---

## H. Agent Permissions

Explicit human approval is required **before** an agent may:

- Merge pull requests.
- Publish releases.
- Delete repositories, issues, or important data.
- Change repository permissions.
- Perform destructive infrastructure operations.

Everything else (branches, commits, pushes, Issues, Projects updates, PRs) is permitted when authorised in the task context.

---

## I. Approved Decisions

Decisions ratified by the project owner are recorded as ADRs in `docs/decisions/`. Currently approved:

- **ADR-0001 (Accepted):** Orbit technology stack — Go + Chi + Docker Engine (official Go client) + SQLite (`modernc.org/sqlite`) + Cobra + OpenAPI + SSE + React/TS/Vite/Tailwind/shadcn dashboard (sequenced after the MVP API). Implementation must follow [docs/features/orbit/design/implementation-plan.md](docs/features/orbit/design/implementation-plan.md) in phase order.

When an approved ADR conflicts with older design text, the ADR wins and the design document should be updated.
