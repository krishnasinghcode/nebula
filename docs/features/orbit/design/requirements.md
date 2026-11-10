# Orbit — Requirements

> Status of items below: **Proposed** until approved. This document defines *what* Orbit must do; *how* is in [technical-design.md](technical-design.md) and [architecture.md](architecture.md). Priorities are in [feature-priorities.md](feature-priorities.md).

## 1. Product Goal (recap)

Orbit is Nebula's EC2-inspired compute service. It must let a developer on an ordinary laptop provision, configure, connect to, and run real workloads in cloud-like compute instances. The goal is a **usable local cloud compute environment that teaches real cloud concepts** — not a Docker wrapper with a dashboard, and not an AWS imitation.

The defining test for every requirement: *does this make Orbit behave like a cloud, or merely like Docker?*

## 2. Functional Requirements

### 2.1 Instance Lifecycle

| ID | Requirement |
|---|---|
| FR-L1 | Create an instance from a named image with a chosen instance type; the instance reaches a `running` state asynchronously, with progress observable. |
| FR-L2 | List instances (summary) and describe an instance (detail: state, type, image, addresses, age, error info). |
| FR-L3 | Start and stop instances; a stopped instance persists its data and can be started again. |
| FR-L4 | Reboot a running instance. |
| FR-L5 | Terminate an instance; termination is irreversible after a confirm step and releases resources (compute, ports, volumes unless flagged keep). |
| FR-L6 | Instance state reflects reality: if the underlying runtime dies, Orbit detects it and updates state rather than showing a stale `running`. |
| FR-L7 | All lifecycle operations are idempotent (repeat requests do not create duplicates or fail confusingly). |

### 2.2 Images

| ID | Requirement |
|---|---|
| FR-I1 | A curated catalogue of Orbit images (Debian, Ubuntu, Alpine, Fedora) with an SSH server pre-configured — the "AMI" of Orbit. |
| FR-I2 | Create an instance only from a catalogue image (MVP). Image metadata: id, os, size, description, tags. |
| FR-I3 | (Later) Users can register their own images from a Dockerfile/image tarball, with a trust warning. |

*Why curated images rather than arbitrary Docker Hub images: an Orbit image must contain an SSH server, sane defaults, and cloud-init-like bootstrap behaviour. Arbitrary images would silently break the "connect like a VM" experience.*

### 2.3 Resource Allocation

| ID | Requirement |
|---|---|
| FR-R1 | Predefined **instance types** (e.g., `orbit.nano` 0.5 vCPU / 512 MB, `orbit.micro` 1 / 1 GB, `orbit.small` 2 / 2 GB) — cloud-style flavours, not raw Docker flags. |
| FR-R2 | Enforced CPU and memory limits per instance at the runtime level. |
| FR-R3 | Provisioning requests are validated against available host capacity plus a reserved host buffer; over-commit requests are rejected with a clear error. |
| FR-R4 | Platform-wide quotas: max running instances, max total memory, max total disk — defaults configurable. |
| FR-R5 | Host out-of-memory / disk-full situations produce actionable errors, not silent corruption. |

### 2.4 SSH Access

| ID | Requirement |
|---|---|
| FR-S1 | Users create named SSH key pairs in Orbit (public key upload or generated keypair). |
| FR-S2 | Instances are launched referencing a key; the key's public part is injected into the instance at first boot. |
| FR-S3 | `orbit ssh <instance>` (or documented equivalent) connects over **real SSH** using the selected key — no emulated terminal. |
| FR-S4 | Lost/deleted keys: no host-side password backdoor; the documented recovery path is re-creating the instance (mirrors real cloud behaviour, teaches real habits). |

### 2.5 Networking

| ID | Requirement |
|---|---|
| FR-N1 | Every instance gets a stable **private IP** on an Orbit-internal network; instance-to-instance traffic works and is addressable by IP or name. |
| FR-N2 | Users can publish selected instance ports to the host (cloud-style "open port" action) with explicit host-port assignment or auto-allocation. |
| FR-N3 | Unpublished ports are not reachable from outside the host (deny by default). |
| FR-N4 | (P1) Security-group-style rules to control instance-to-instance traffic. |

### 2.6 Storage

| ID | Requirement |
|---|---|
| FR-D1 | Each instance has a persistent root volume: data survives stop/start, and is destroyed on terminate (explicit, like real cloud root volumes). |
| FR-D2 | Users can request extra storage size at creation (within quota); a thin disk limit is enforced where the runtime supports it. |
| FR-D3 | (P1) Attach/detach additional named volumes. |

### 2.7 Configuration / Startup Scripts

| ID | Requirement |
|---|---|
| FR-C1 | User data: a startup script (shell) executed on first boot — cloud-init-like, simplified. |
| FR-C2 | Instance tags/metadata (name, key-value tags) settable at creation and readable from inside the instance via a metadata endpoint or file (P1 for in-guest endpoint). |

### 2.8 Monitoring & Logs

| ID | Requirement |
|---|---|
| FR-M1 | Instance status view: state, uptime, addresses, resource limits. |
| FR-M2 | (P1) CPU/memory/network utilisation per instance from runtime metrics. |
| FR-M3 | (P1) Access instance console logs (boot output). |

### 2.9 Developer Experience

| ID | Requirement |
|---|---|
| FR-X1 | A CLI covering all MVP operations (primary interface). |
| FR-X2 | JSON output mode for scripting (`--output json`). |
| FR-X3 | Clear, cloud-familiar error messages (e.g., quota exceeded, insufficient capacity, key not found). |
| FR-X4 | (P1) Minimal web dashboard for list/create/connect workflows. |

## 3. Non-Functional Requirements

| ID | Requirement |
|---|---|
| NFR-1 | **Simplicity:** single-host, single-binary control plane; no required external services beyond Docker. |
| NFR-2 | **Laptop-friendly:** default quotas leave headroom (see [data-and-api.md defaults] and [technical-design.md §Local resources]); total default footprint ≤ ~50 % of host RAM. |
| NFR-3 | **Fast feedback:** create-to-SSH-ready for a small image ≤ 60 s on a typical laptop. |
| NFR-4 | **Honesty:** the UI/docs never imply VM-grade isolation or multi-tenant security. |
| NFR-5 | **State durability:** control-plane state survives Nebula restarts; instances are recovered/reconciled after host reboot. |
| NFR-6 | **Portability:** runs on Docker Desktop (Windows/macOS) and native Docker Engine (Linux), with documented behavioural differences. |
| NFR-7 | **Evolvability:** the runtime is behind an interface so a QEMU/KVM backend can be added without redesigning the control plane. |

## 4. User Stories

- **As a learner**, I can create a "small" instance from Ubuntu in one command and SSH into it, so I can practise working with a server like I would on EC2.
- **As a developer**, I can pass a startup script that installs my app and starts it, so my instance is ready on boot.
- **As a developer**, I can publish port 8080 of an instance to my laptop, so I can open my web app in a browser.
- **As a developer**, I can run three instances that talk to each other over private IPs, so I can prototype distributed systems.
- **As a learner**, I can stop an instance and find my files still there tomorrow, and terminate it when I'm done, so I learn real volume semantics.
- **As a cautious user**, I can see how much host CPU/memory/disk my instances are allowed to consume, so Orbit never eats my whole laptop.
- **As a learner**, when my instance crashes, I see it in a `crashed/failed` state with a reason — not a lie about `running`.

## 5. Edge Cases and Failure Scenarios

| Scenario | Required behaviour |
|---|---|
| Host reboots while instances run | On Nebula start, reconcile: containers are gone/exited → instances move to `stopped` (or `failed`), never stale `running`. |
| Docker daemon stops mid-operation | Operation errors are surfaced; state stays consistent; retriable. |
| Instance OOM-killed | Runtime reports non-zero exit / OOM; state → `failed` with reason. |
| Duplicate create (double submit) | Idempotency token or unique-name rule returns the existing instance. |
| Terminate while provisioning | Provisioning is cancelled; instance reaches `terminated` without leaking containers/volumes. |
| Request exceeds capacity | Rejected up-front with numbers (requested vs available). |
| Port already in use on host | Clear error; no partial instance. |
| Host disk full during volume write | Instance errors; quota validation reduces likelihood; actionable message. |
| Deleting an SSH key in use | Blocked or explicit force-with-warning; never silently breaks running instances' future boots. |
| Clock/user changes, VPNs, firewalls on host | Documented per-OS caveats (see [technical-design.md §Local resources]); Orbit fails with diagnostics rather than hanging silently. |
| Concurrent operations on one instance | Serialized per-instance (single active transition); conflicting requests get a 409. |

## 6. Explicit Non-Requirements (this phase)

- Multi-tenant/multi-user accounts and RBAC (single local user in MVP).
- Autoscaling, load balancing, regions/AZs, placement groups.
- VM-grade isolation claims (see [security.md](security.md)).
- Parity with the full EC2 API surface.

## 7. Open Questions

1. Web dashboard in P1 or pushed to P2? (CLI-first is assumed.)
2. Should `orbit ssh` fall back to `docker exec` when an image lacks `sshd` (convenience) or always require real SSH (honesty)? **Recommendation: require real SSH; refuse with a clear message.**
3. Default host resource reserve percentages — confirm after measuring real Docker Desktop overhead on Windows/macOS.
