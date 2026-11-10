# Orbit — Feature Priorities

> Priorities are **Proposed**. Dependencies reference [requirements.md](requirements.md) FR ids. The rule applied: P0 is the minimum set that makes Orbit *a usable local cloud compute environment running real workloads*, not a Docker wrapper.

## Priority Definitions

- **P0 — MVP:** required for a usable first release; without it Orbit fails its core promise.
- **P1 — Next:** significantly improves realism/UX; next development cycle.
- **P2 — Advanced:** differentiating/advanced; implement when the foundation is proven.
- **Out of scope:** not worth building — unnecessary, impractical, or contradicts Nebula's goals.

## P0 — MVP

| Feature | Why P0 | Depends on |
|---|---|---|
| Instance create/start/stop/reboot/terminate/list/describe (FR-L1–L7) | The core cloud experience; everything else hangs off it. | — |
| Curated images with SSH pre-configured (FR-I1–I2) | Guarantees the "connect like a VM" experience; arbitrary images would break it. | Lifecycle |
| Instance types with enforced CPU/memory limits (FR-R1–R2) | Cloud-style abstraction + laptop protection; without limits one instance can starve the host. | Lifecycle |
| Host capacity validation + basic quotas (FR-R3–R4) | NFR-2: Orbit must never eat the laptop; also teaches resource planning. | Types |
| SSH key pairs + real SSH access (FR-S1–S4) | The single most "this feels like a real cloud" feature; `docker exec` does not teach SSH or keys. | Images |
| Private network + per-instance IP + instance-to-instance connectivity (FR-N1) | Enables real distributed workloads — the key differentiator from a Docker wrapper. | Lifecycle |
| Port publishing (deny-by-default) (FR-N2–N3) | Users must run actual applications (web servers etc.) reachable from the host. | Network |
| Persistent root volume per instance (FR-D1) | Stop/start keeping data is fundamental cloud semantics. | Lifecycle |
| Startup script / user data (FR-C1) | Lets users deploy and start real apps at boot — "run actual workloads". | Lifecycle |
| CLI covering all of the above (FR-X1–X3) | Primary interface; JSON output enables scripting. | All |
| State reconciliation after host/Nebula restart (FR-L6, NFR-5) | Without it the platform lies about state after every reboot. | Lifecycle |
| Core security hardening (non-privileged containers, dropped capabilities, no socket mount, resource limits) | Non-negotiable before exposing instance shells to users. See [security.md](security.md). | All |

**The MVP test:** a user can `orbit instance create ubuntu --type orbit.micro --key dev --user-data app.sh`, SSH in, run a web app, reach it from the host browser, have a second instance talk to the first over a private IP, stop both, and find everything intact tomorrow. That is Orbit, not a Docker wrapper.

## P1 — Next

| Feature | Why P1 | Depends on |
|---|---|---|
| Per-instance metrics (CPU/mem/net) and console logs (FR-M2–M3) | High teaching value ("what is my instance doing"), cheap via runtime stats API. | Lifecycle |
| Additional named volumes, attach/detach (FR-D3) | Teaches EBS-style block storage semantics distinctly from root volumes. | Lifecycle |
| Security-group-style traffic rules (FR-N4) | Teaches network policy; needs stable network foundation first. | Network |
| In-instance metadata endpoint (FR-C2) | Teaches IMDS concepts; useful for bootstrap scripts. | Network |
| User-supplied images with trust warnings (FR-I3) | Power-user need; risks breaking SSH/boot assumptions without guardrails. | Images |
| Minimal web dashboard (FR-X4) | Big UX win, but only worthwhile once the API is stable. | API |
| Tags and simple filtering | Organisational hygiene at small scale. | Lifecycle |

## P2 — Advanced

| Feature | Why P2 | Notes |
|---|---|---|
| Snapshots and restore | High value (teaches AMI/backup concepts) but needs a working volume model first. | [technical-design.md §Storage] |
| Infrastructure-as-Code (declarative Orbit configs) | Differentiating; depends on a stable, complete API. | [testing-strategy.md] consumers |
| Network failure/latency simulation | Genuinely differentiating for distributed-systems learning; non-trivial on Docker Desktop (tc/netem availability varies by OS). | Depends on Linux-native networking |
| Cost estimation (simulated pricing) | Fun and educational, but artificial; wait for a real usage/metrics layer. | Metrics |
| Multi-instance templates / small environments ("stacks") | Useful once single-instance flows are proven. | IaC |
| QEMU/KVM runtime backend | The real isolation upgrade path; architecture must leave room for it (see [architecture.md §Evolution]). | Runtime abstraction |

## Out of Scope

| Feature | Reason |
|---|---|
| Multi-tenant isolation, accounts, RBAC, billing | Orbit is a local single-user learning platform; see [security.md §Trust model]. |
| VM-grade isolation *claim* (before a VM backend exists) | Dishonest with container runtime; handled by trust-model labelling instead. |
| Autoscaling, load balancers, regions/AZs | Meaningless on one laptop; would imitate AWS without teaching anything extra. |
| Full EC2 API parity (hundreds of operations) | Explicitly against Nebula's "practical over complete" principle. |
| Windows containers, nested containers, GPU instances | Complexity/hardware requirements out of proportion to value. |
| Managed orchestration (Kubernetes-in-Nebula) | A different product; would consume all effort. |

## Anti-Goal Check

Features were rejected when they (a) imitate AWS without adding learning value, (b) require infrastructure a laptop doesn't have, or (c) promise security the container runtime cannot deliver. See [security.md] for the honest statement of what the MVP security boundary is and is not.
