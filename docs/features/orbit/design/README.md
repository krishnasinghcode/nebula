# Orbit — Technical Design Package

Design specification for Nebula Orbit. **Status: approved and implemented (MVP, v0.1.0).** Individual documents may retain pre-implementation status lines ("Proposed") in their headers; the authoritative record of what shipped, deviations, and open questions is [implementation-plan.md](implementation-plan.md).

| Document | Contents |
|---|---|
| [requirements.md](requirements.md) | Functional & non-functional requirements, user stories, edge cases |
| [feature-priorities.md](feature-priorities.md) | P0/P1/P2/out-of-scope with justification |
| [architecture.md](architecture.md) | Components, control/execution planes, flows, Mermaid diagrams, evolution path |
| [technical-design.md](technical-design.md) | Implementation approach per capability, technology choices, trade-offs |
| [security.md](security.md) | Trust model, threat model, isolation, least privilege, security gates |
| [data-and-api.md](data-and-api.md) | Data model, lifecycle state machine, API contracts |
| [dashboard-design.md](dashboard-design.md) | Dashboard UI design system: themes, tokens, console patterns (keep new UI consistent) |
| [testing-strategy.md](testing-strategy.md) | Test layers, scenario catalogue, measurable acceptance criteria |
| [implementation-plan.md](implementation-plan.md) | **Execution plan for agents**: phase checklists, test gates, resume protocol, progress tracking |

---

# Final Recommendation

## 1. Recommended MVP Feature Set

Instance lifecycle (create/start/stop/reboot/terminate/list/describe, idempotent, reconciled) · curated SSH-enabled images · three instance types with enforced CPU/memory limits · admission control with host reserve + quotas · SSH key pairs and real SSH access · one bridge network with private IPs and instance-to-instance connectivity · deny-by-default port publishing (loopback default) · persistent root volume per instance · shell user-data bootstrap · CLI with JSON output · hardening profile from [security.md §8] as a hard gate.

The MVP test: create → SSH → run a web app → reach it from the host → second instance talks to it over a private IP → stop → data intact tomorrow. That is a local cloud, not a Docker wrapper.

## 2. Recommended Architecture

Single-host **modular monolith** control plane (API, instance manager, scheduler/capacity, network/storage/key managers, reconciler, SQLite state) behind a **runtime adapter interface**; execution plane = Docker Engine running hardened containers. No microservices, no message broker, no guest agent. The adapter seam is the prepared path to a QEMU/KVM backend.

## 3. Initial Technology Choices (proposed, with justification)

- **Go** — first-class Docker SDK, single static binary for CLI + control plane, robust concurrency for the reconciler.
- **Docker Engine API via official SDK** (pinned minimum API version) — the stated expectation; portable across Linux/macOS/Windows.
- **SQLite (WAL)** — durable, transactional, zero-ops state at single-user scale.
- **REST/JSON on localhost** — scriptable, UI-ready; gRPC unjustified for one host.
- **Real sshd in curated images + Ed25519 keys** — teaches the real thing; no agent.
- **Custom bootstrap entrypoint + shell user-data** — cloud-init's learning value without its weight.

## 4. Security Requirements Before First User Exposure

From [security.md §8], non-negotiable: no privileged containers / caps dropped / no-new-privileges / seccomp; no docker.sock or host mounts or host namespaces (validated + tested); deny-by-default networking with loopback-default port publishing; token-authenticated loopback-only API; mandatory resource limits; admission control + safe quotas; and the **trust-model statement** shown to users — Docker containers are not VM-grade isolation, and untrusted code should wait for the VM backend.

## 5. Major Technical Risks

1. **Shared-kernel isolation** — mitigations raise the bar; only a VM runtime truly closes it. Highest-severity, known-limitation risk.
2. **Docker Desktop divergence** (macOS/Windows): VM-based capacity, no direct private-IP reachability, slower volumes, firewall quirks. Mitigation: treat Engine-reported resources as the envelope; port publishing as the portable connectivity path; per-OS test profiles (A2).
3. **State/reconciliation correctness** — the reconciler is the component most likely to produce lying state. Mitigation: integration suite with a scriptable failure-injecting adapter; invariant tests.
4. **Disk quota enforcement gap** in MVP — admission-level only until P1.
5. **Curated-image maintenance** — images must keep working across Docker/base-image updates; needs a small build/test pipeline early.

## 6. Implementation Sequence

1. **Runtime adapter + skeleton control plane** (create/start/stop/remove with hardcoded limits) — proves the riskiest seam first.
2. **State store + state machine + reconciler** — honest state before features.
3. **Curated images + SSH key manager + SSH access** — the core experience.
4. **Instance types + capacity/quota admission control.**
5. **Networking: bridge network, private IPs, port publishing.**
6. **User-data bootstrap + root volumes.**
7. **CLI polish, error messages, JSON output.**
8. **Security gates + E2E acceptance suite (A1–A7).**
9. *Then* P1: metrics/logs, extra volumes, security groups, metadata endpoint, dashboard.

## 7. Decisions Requiring Approval Before Development

~~1–8 below~~ **Status: the technology stack (former items 1 and 7) and the Docker runtime are now APPROVED — recorded as [ADR-0001](../../../decisions/0001-orbit-technology-stack.md). Remaining approvals:**

2. **Curated-image-only MVP** (no arbitrary user images until P1, with trust warnings).
3. **Real-SSH-only policy** — no `docker exec` fallback ([requirements.md §7 Q2]).
4. **Single shared bridge network in MVP** (full instance-to-instance reachability, documented) vs. per-group networks now.
5. **Default quotas and host reserve** (25 % RAM / 1 core reserve; 5 instances / 8 GB / 20 GB) — confirm after measuring target hardware.
6. **Security gate list [security.md §8] as a release-blocking definition of done.**
8. **Async create/stop semantics** (202 + reconcile model) vs. synchronous responses.

**Unresolved questions carried from requirements.md:** dashboard timing (P1 vs P2); user-data re-run semantics on restart (once-only recommended); exact default reserve percentages pending measurement.
