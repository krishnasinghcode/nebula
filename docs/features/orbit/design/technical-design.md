# Orbit — Technical Design

> **Status: Proposed.** For each capability: mechanism → recommendation → technologies/APIs → alternatives and trade-offs → limitations. Technology choices here are recommendations pending approval (see the design README's decision list).

Related: [architecture.md](architecture.md) · [security.md](security.md) · [data-and-api.md](data-and-api.md)

## 1. Containers and Lifecycle Management

**Mechanism.** Docker Engine exposes a REST API over a Unix socket (Linux) or named pipe (Windows). Containers are processes in namespaces/cgroups; lifecycle = create → start → stop (SIGTERM, then SIGKILL after timeout) → remove.

**Recommendation.** Orbit drives the Engine API directly with the official Docker SDK (Go client). Each Orbit Instance is one container with: `--restart=no` (Orbit's reconciler owns restart semantics — never let Docker resurrect instances), explicit `--stop-timeout` (graceful stop, then kill), labels (`orbit.instance-id`, `orbit.image`, …) for ownership/orphan detection.

**Alternatives.**
- *CLIs (`docker` binary) via shelling out:* brittle parsing, harder errors, no streaming stats. Rejected.
- *containerd directly:* fewer layers, but we'd re-implement networking/volume plumbing Docker gives us. Revisit only if Docker Desktop limits hurt (see §12).

**Limitations.** Docker version drift across platforms (Docker Desktop vs native) affects some API fields; pin a minimum Engine API version. Container exit codes conflate app crashes and OOM — OOM is detected via the API's state flags (`OOMKilled`), which we map to `failed` + reason.

## 2. Images and Instance Templates

**Mechanism.** Docker images are layered, content-addressed filesystems; a container's root filesystem is ephemeral *unless* backed by a volume.

**Recommendation.** A small **curated catalogue** of Orbit images (Debian, Ubuntu, Alpine, Fedora) built from simple Dockerfiles that add: `sshd` (configured for key-only auth, port 22), a bootstrap entrypoint that (a) injects the user's public key, (b) writes user-data to a known path and executes it once, (c) starts sshd last. Images are ordinary Docker images; `orbit` maps image names to them. Root filesystem is **read-only container filesystem + writable volume** for persistence (§5).

**Alternatives.**
- *Allow arbitrary Docker Hub images:* breaks SSH/boot guarantees. Deferred to P1 with trust warnings and validated entrypoint wrapping.
- *Build VM images (qcow2) now:* pointless until a VM backend exists; keep images behind an abstraction.

**Limitations.** Curated images must be built and versioned by the project; first `create` triggers an image pull (tens of MB–hundreds of MB) — the 60 s create-to-SSH target (NFR-3) assumes the image is cached.

## 3. CPU and Memory Restrictions

**Mechanism.** Docker sets cgroup (v2) limits: `--cpus` (CPU quota via `cpu.max`) and `--memory` (+ `--memory-swap=--memory` to disable swap borrowing so limits are honest). Memory over-commit → cgroup OOM-kill.

**Recommendation.** Instance types map directly to these flags: `orbit.nano` = 0.5 CPU / 512 MB, `orbit.micro` = 1 / 1 GB, `orbit.small` = 2 / 2 GB. Always set `--memory-swap` equal to `--memory`. CPU is a *quota*, not a pin (no `--cpuset` — a laptop shares cores anyway).

**Alternatives.** Pinning cores (`--cpuset-cpus`) gives predictable benchmarks but fragments capacity and fights laptop scheduling. Rejected.

**Limitations.** On Docker Desktop, containers run inside the Desktop VM — limits are relative to the *VM's* allocation, not the physical host. Capacity detection must account for this (§12). cgroup limits require the cgroup controller to be available (it is, in all supported setups).

## 4. Persistent Volumes

**Mechanism.** Docker volumes are host directories managed by Docker; bind mounts map arbitrary host paths; both survive container removal.

**Recommendation.** One **named Docker volume per instance** as the root data volume, mounted at a fixed location; the container's own filesystem holds only the OS (and can be considered disposable). On terminate, the volume is deleted unless the user opts to keep it. Named volumes (not bind mounts) are used because they're Docker-managed, avoid host-path permission mess on Linux (UID mapping), and are portable to other runtimes behind the adapter.

**Alternatives.**
- *Bind mounts under a Nebula data dir:* easier to browse from the host, but host-UID/permission issues on Linux and leaks host paths into the guest.
- *Compose or named container filesystem:* no persistence or awkward lifecycle.

**Limitations.** Volume *size quotas* are not natively enforced for local volumes (no thin provisioning); see Storage quotas in [security.md §Resource protection] for the pragmatic mitigation. Root-volume *requested size* is honoured as best-effort in MVP (documented), with real disk-limit enforcement a P1 item (per-instance disk image or filesystem quota via project quotas on the data dir).

## 5. SSH Server Provisioning and Key Management

**Mechanism.** Real SSH = an `sshd` daemon inside the instance authenticating with public keys. Key injection happens at first boot via the image's bootstrap entrypoint; the host never logs into the guest with a password.

**Recommendation.**
- Users register named keys (`orbit key create dev` generates an Ed25519 pair client-side; or upload a public key). Only the **public key** is stored server-side; the private key stays in the user's `~/.orbit/keys` with `0600` perms. Ed25519: modern, small, fast.
- At create, the instance references a key id; bootstrap injects `authorized_keys` for the instance's non-root user and disables password auth entirely (provisioned via `-o PasswordAuthentication=no`).
- `orbit ssh <instance>` wraps plain `ssh -i <private key> orbit-user@<reachable-address>`. It resolves which address/port to use (published port on Docker Desktop; private IP on Linux — §7). No agent daemon.

**Alternatives.**
- *`docker exec` "SSH":* convenient but teaches nothing about keys/hosts and is dishonest about the cloud experience. Rejected as the primary path (open question in requirements.md recommends refusing with a clear message when an image lacks sshd).
- *Host-side SSH proxy/jump host:* extra moving part; unnecessary at this scale.

**Limitations.** Key must be injected at *first* boot only (adding keys to a running instance is out of MVP scope — mirrors EC2's replace-key semantics being non-trivial). Losing the private key = locked out (by design, FR-S4).

## 6. Private IP Allocation and Networking

**Mechanism.** Docker bridge networks give each container an IP from a subnet, with container-to-container connectivity and NAT for outbound traffic.

**Recommendation.** Orbit creates **one managed bridge network** (`orbit-net`, e.g. `172.28.0.0/16` — a range unlikely to collide with common home/office LANs; configurable). Every instance joins it; Docker assigns the private IP, which Orbit records as the instance's `private_ip`. Instance-to-instance traffic works by IP; DNS aliases (container name) give friendly names. Outbound internet access works via Docker's NAT (useful for installing packages — a core learning use case).

**Alternatives.** Per-tenant/user networks, macvlan (direct LAN presence — complicates security and portability), overlay networks (multi-host only). All deferred; one bridge is the simplest thing that satisfies FR-N1.

**Limitations.** On Docker Desktop, "the host" for private IPs is the Desktop VM; container IPs are not directly reachable from the laptop's LAN stack (hence §7). Subnet collisions with corporate VPNs are possible; documented + configurable.

## 7. Port Mapping and Firewall Rules

**Mechanism.** Docker's userland proxy + iptables rules implement `-p <host>:<guest>` publishing; unpublished ports are unreachable from outside the host (Docker's default).

**Recommendation.** "Publish a port" is an explicit Orbit operation (`orbit instance expose <id> --port 8080`), allocating a host port (auto or user-chosen) and mapping to the instance port; mappings are recorded in the state store and shown in `describe`. Deny-by-default comes free with Docker's model. No firewall manipulation of the host in MVP.

**Alternatives.** Security-group-style rule engine (iptables/nftables directly) = the P1/P2 SG feature; doing raw iptables in MVP is fragile across Docker Desktop (rules live inside the Desktop VM). Deferred deliberately.

**Limitations.** Host-published ports are bound on the laptop and follow host firewall rules — users must understand these are *host* ports (this is also true in real clouds with public IPs, so it's teachable). See [security.md §Networking] for what this exposes.

## 8. Startup Scripts (cloud-init-like)

**Mechanism.** cloud-init is a guest-side bootstrap framework (instance metadata + per-stage modules). Too heavy to embed for our needs.

**Recommendation.** **Simplified user data:** the image's bootstrap entrypoint writes the user-data payload to `/var/lib/orbit/user-data` and runs it once (shell script in MVP; log to `/var/log/orbit/bootstrap.log` for debugging). It runs as the instance's admin user with sudo. An executed-once marker prevents re-running on restart. Metadata (instance id, type, tags) is exposed as a local file in MVP; an HTTP metadata endpoint is P1 (FR-C2).

**Alternatives.** Real cloud-init: supports many distros out of the box and teaches the exact same concept by name — but adds guest complexity, slower boot, and harder debugging; our script model covers 90 % of the learning value. Revisit if images grow.

**Limitations.** Shell scripts only in MVP; no YAML/cloud-config syntax. Boot is not "finished" until the script exits — the reconciler can expose `booting` → `running` accordingly (simple, honest).

## 9. Instance Metadata

**Recommendation.** The state store is authoritative. Instance record fields and the metadata exposure (file now, endpoint later) are specified in [data-and-api.md]. No sidecar metadata database.

## 10. Monitoring, Logs, Metrics

**Mechanism.** Docker Engine API provides `stats` (streaming CPU/mem/net/blkio per container) and `logs` (stdout/stderr).

**Recommendation.** MVP: `describe` shows state/uptime/limits; `orbit logs` tails container logs; `orbit top` (P1) samples the stats stream on demand — **no continuous metrics collection or TSDB in MVP** (overkill). Console boot output is available via the bootstrap log. P2 could add periodic sampling into SQLite if there's demand for simple graphs.

**Limitations.** No historical metrics in MVP (deliberate — see feature-priorities.md). Stats inside Docker Desktop reflect the Desktop VM's accounting.

## 11. State Persistence and Recovery

**Mechanism.** SQLite gives durable, transactional, zero-ops local state; the reconciler (architecture.md §4.3) reconciles records with runtime reality.

**Recommendation.** Single SQLite file in the Nebula data dir, WAL mode. All state transitions are transactions. On control-plane start: reconcile all non-terminal instances before serving API (short startup delay, honest state). Idempotent operations + serialized transitions make retries safe.

**Limitations.** SQLite is single-writer — fine for one user; fine forever for Orbit's scope. Backups of the state file are a docs note, not a feature.

## 12. Local Resource Management (host capacity, OS differences)

**Detection.** Read `HostConfig` resources from the Docker Engine API (`NCPU`, `MemTotal`, backing filesystem stats for the Docker data root). On Docker Desktop these are the **Desktop VM's** values — treat them as the real capacity envelope, because containers are limited by them anyway (this automatically handles "user gave Desktop 4 GB" cases).

**Recommendation (defaults, configurable):**
- Reserve ~25 % of detected memory + ~1 core for the host; schedulable budget = total − reserve (per [requirements.md NFR-2]).
- Quotas: max 5 running instances, max 8 GB total instance memory, max 20 GB total volumes (all defaults, user-tunable).
- Admission control at create: sum(allocated) + request ≤ budget and ≤ quota → else reject with requested-vs-available numbers.
- Overcommit: CPU overcommit allowed (quotas, not reservations — matches cloud bursting semantics); **memory overcommit never** (OOM-killed instances teach the wrong lesson).

**Per-host differences.**
- *Linux, native Engine:* full fidelity — private IPs reachable from host, best metrics, cgroups v2.
- *macOS/Windows, Docker Desktop:* containers live in a VM; capacity = VM config; private IPs not directly host-reachable (publish ports instead); fileshare/volume performance is slower; features like `tc` netem may be unavailable (affects P2 network simulation).
- *Windows hosts:* containers are Linux containers inside WSL2; `orbit ssh` targets `localhost:mappedPort`. Windows-native containers: out of scope.

**Failure behaviour.** Documented + validated up front (NFR-2, FR-R5): reject at admission; runtime-level exhaustion surfaces as `failed` + reason via the reconciler.

## 13. Technology Summary (proposed)

| Concern | Choice | Why |
|---|---|---|
| Language | **Go** | Official Docker SDK is first-class; single static binary for CLI + control plane; strong concurrency for the reconciler. |
| Runtime | **Docker Engine API** (official SDK, pinned minimum API version) | The stated expectation; de-facto standard; portable across all three OSes. |
| State | **SQLite (WAL)** | Zero-ops, durable, transactional — right size for single-host. |
| API | **REST/JSON over localhost HTTP** | Simple, scriptable, UI-ready; gRPC unnecessary for one host. |
| SSH | **Real sshd in curated images; Ed25519 keys; Orbit wraps `ssh`** | Teaches the real thing; no agent required. |
| Bootstrap | **Custom entrypoint + user-data script** | cloud-init's value without its weight. |

Rejected-for-now: Kubernetes/everything-on-K8s (orders of magnitude of complexity for zero user value here), containerd direct (re-implementation cost), shelling out to Docker CLI (fragility), embedded VM hypervisors in MVP (deferred to the runtime-abstraction evolution path).
