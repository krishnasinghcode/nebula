# Orbit — Security, Isolation and Least Privilege

> **Status: Proposed.** Every measure states: threat addressed, implementation, MVP or later, and residual limitations. Cross-references: [technical-design.md](technical-design.md) explains mechanisms; [feature-priorities.md](feature-priorities.md) tracks priorities.

## 1. Trust Model — Read This First

**Orbit with the container runtime is NOT a secure multi-tenant cloud hypervisor, and must never be described as one.**

Containers share the host kernel. A container root with sufficient privileges, a kernel vulnerability, or a misconfigured mount can lead to host compromise. Docker's default hardening is good, but it is categorically weaker than hardware-virtualised isolation (QEMU/KVM).

Recommended boundary:

| Workload | Verdict |
|---|---|
| Your own learning workloads, dev tools, web apps you wrote | **Trusted enough** for the container runtime. This is the intended use. |
| Running untrusted third-party software "to see what it does" | **Not safe** on the container runtime. Wait for the QEMU/KVM backend, or use an external VM. |
| Malicious adversaries with shell access, deliberately attacking the host | **Out of threat model for MVP.** Defences below raise the bar; they do not close it. |

Product honesty requirement (NFR-4): Orbit surfaces this statement in docs and at instance-creation time (a one-time acknowledgement in MVP). The long-term fix is a real VM runtime behind the runtime adapter ([architecture.md §Evolution]).

## 2. Threat Model (MVP)

| # | Threat | Vector | Mitigation (section) |
|---|---|---|---|
| T1 | Container escape to host | Kernel exploit from inside instance | §3.1 hardening, §1 honesty |
| T2 | Host filesystem access | Mounts, Docker socket, `/var/run/docker.sock` exposure | §4 |
| T3 | Instance-to-instance attack | Bridge network access | §5 |
| T4 | Host service exposure via published ports | Port publishing binds on host | §5.3 |
| T5 | Unauthorized management API use | Local process talking to Orbit API | §6 |
| T6 | Resource exhaustion / denial of service against the host | Fork bombs, memory hogs, disk fillers | §7 |
| T7 | Key theft / credential misuse | Private keys on disk, key sharing | §6.2 |
| T8 | Privilege escalation inside instance | User-data runs as sudo | §3.2, §6.3 |

## 3. Container Isolation

### 3.1 Container hardening profile (every Orbit instance)

| Measure | Implementation | Threat | MVP? | Residual limitation |
|---|---|---|---|---|
| **No privileged containers** | Never set `Privileged: true`; adapter rejects it | T1 | ✅ P0 | — |
| **Drop all capabilities, add back the minimum** | `CapDrop: ALL` (+ `CHOWN,NET_BIND_SERVICE,SETUID,SETGID,SYS_TIME`-class set only where images need it); `--no-new-privileges` | T1, T8 | ✅ P0 | Minimum set must be validated per curated image |
| **Non-root guest user** | Curated images default to a `orbit` admin user with passwordless sudo only for documented bootstrap needs; root login disabled | T1, T8 | ✅ P0 | sudo inside guest is a learning trade-off; see §6.3 |
| **Seccomp** | Docker's default seccomp profile (blocks ~44 syscalls, restricts many more) | T1 | ✅ P0 (default) | Custom profiles are P2; default is sufficient |
| **Read-only root filesystem** | `ReadonlyRootfs: true`; writable paths via the data volume and `/tmp` tmpfs | T1, T2 | ✅ P0 | Some packages need extra writable dirs — handled per curated image, documented |
| **User namespaces (rootless)** | Prefer running the Docker Engine in **rootless mode** when available; else rely on the above + separate userns mapping where supported | T1 | ⚠️ Best-effort P0/P1 | Rootless Docker changes networking/ports behaviour (esp. Docker Desktop); treat as recommended-when-possible, not required |
| **AppArmor/SELinux** | Keep default Docker profiles on Linux (AppArmor on Ubuntu/Debian, SELinux on Fedora) | T1 | ✅ P0 (default, Linux only) | Not available via Docker Desktop macOS; Windows depends on WSL2 |

### 3.2 What this buys and what it doesn't

With the P0 profile, an instance cannot: load kernel modules, mount host paths, access the Docker socket, use most exotic syscalls, or write to its system root. It *can* still execute syscalls against the shared host kernel — a 0-day kernel exploit remains the primary escape path. **This is exactly the gap the future QEMU/KVM backend closes** ([architecture.md §Evolution]).

## 4. Host Protection

| Measure | Implementation | MVP? |
|---|---|---|
| **Never expose the Docker socket to instances** | Adapter refuses any create spec containing docker.sock mounts, privileged mode, or `PidMode: host`/`NetworkMode: host`/`IpcMode: host`. Enforced in code, plus validation tests. | ✅ P0 |
| **No host bind mounts in MVP** | Only Docker-managed named volumes; bind-mount support (if ever added) is allow-listed paths only. | ✅ P0 |
| **No host PID/network namespaces** | Blocked at validation (above). | ✅ P0 |
| **Host process visibility** | Instances see only their own PID namespace; nothing to do beyond keeping it default. | ✅ default |
| **Protect the host's other services** | Deny-by-default port publishing (§5.3); instances reach the host only via the gateway IP (documented, not advertised); no host.docker.internal mounts/proxies unless user-initiated. | ✅ P0 |
| **Control-plane runs least-privilege** | Nebula control plane runs as the user's own account (not root/admin); it only needs access to the Docker socket/named pipe and its data dir. | ✅ P0 |

## 5. Networking Security

### 5.1 Instance-to-instance isolation (T3)

All MVP instances share one bridge network (see [technical-design.md §6]) → **any instance can reach any other instance**. Acceptable for the single-user MVP; documented honestly. Improvement path (P1): per-group networks + security-group rules; the network manager's interface should be designed so network-per-instance or per-group is a configuration change, not a redesign.

### 5.2 Outbound traffic

Instances have internet access via NAT (needed for package installation). P1/P2: optional egress controls via SG rules. No MVP egress filtering — document it.

### 5.3 Port exposure (T4)

- Deny-by-default: only explicitly published ports are reachable from the host.
- Published ports bind to **loopback (`127.0.0.1`) by default**; exposing on all interfaces (`0.0.0.0`) is an explicit user action with a warning — this prevents accidentally serving an instance to the local network (a real risk on shared Wi-Fi).
- Host port ranges restricted to >1024 unless the user explicitly chooses a privileged port (OS may still refuse — fine, surface the error).

### 5.4 Preventing unintended host access

Docker's iptables rules can bypass host firewalls for published ports (ufw/Windows Firewall do not always see them). Mitigation: default loopback binding (above) + documentation. This is a genuine Docker footgun; we accept it with mitigations rather than re-implementing firewalling in MVP.

## 6. Access Management

### 6.1 Management API authn/authz (T5)

Even though the API is localhost-only, **the API requires a bearer token** (generated at first run, stored `0600` in the Nebula data dir). Why: any local process could otherwise create/delete instances and volumes; a token is cheap and makes the future web-UI/multi-user path safe-by-default. Single user, full permissions in MVP. No anonymous operations.

### 6.2 SSH keys (T7)

- Ed25519; private keys generated client-side and never leave the user's machine; server stores public keys + fingerprints only.
- Files `0600`; keys referenced by id, not path; fingerprints shown in `describe` so users can verify what's injected.
- No key-sharing between instances unless the user references the same key (explicit choice, like real clouds).

### 6.3 Inside the instance (T8)

User data scripts run once, as the admin user, with sudo — that's the point of instance bootstrap (real clouds do the same via cloud-init). Residual risk is the user's own script; user-data content is never logged verbatim (paths and exit status only).

### 6.4 Multi-user separation

Out of scope for MVP ([security §1] trust model). The data model keeps an `owner` field on resources now to avoid a painful migration later ([data-and-api.md]).

## 7. Resource Protection (T6)

| Measure | Implementation | MVP? | Limitation |
|---|---|---|---|
| Memory limits (no swap) | cgroup via `--memory` = `--memory-swap` | ✅ P0 | OOM-kill = `failed` state, by design |
| CPU quotas | `--cpus` per instance type | ✅ P0 | Quota ≠ reservation; heavy parallel instances share cores (documented) |
| Process count | `--pids-limit` (e.g. 512 default) | ✅ P0 (cheap, meaningful) | Coarse; per-image tuning |
| File descriptors / ulimits | Sensible defaults via `--ulimit nofile` | ✅ P0 (cheap) | — |
| Disk quotas | **Best-effort in MVP**: total-volume quotas enforced at admission; per-volume enforcement via filesystem quotas is P1 | ⚠️ P0 admission-level | A single instance can still fill its volume until P1 enforcement lands |
| Platform quotas | Max instances / total mem / total disk, enforced at admission ([technical-design.md §12]) | ✅ P0 | — |
| Total in-memory DoS against host | Memory reserve for host (25 % default) | ✅ P0 | Docker Desktop VM sizing also constrains |

## 8. Security Requirements Before First Release (hard gates)

1. No privileged containers; caps dropped; `no-new-privileges`; seccomp default — enforced in the adapter and covered by tests.
2. No docker.sock / host mounts / host namespaces — rejected at validation, regression-tested.
3. Deny-by-default networking; published ports loopback-only by default.
4. Management API token required; loopback-only listener.
5. Resource limits mandatory on every instance (a create without a valid type is invalid).
6. Admission control + quotas active with safe defaults.
7. The trust-model statement ([§1]) appears in user-facing docs and at first create.
8. SSH: password auth disabled in images; key-only; private keys never leave the client.

## 9. Known Residual Risks (documented, accepted for MVP)

1. Shared-kernel container escapes (T1) — mitigated, not eliminated. Long-term: VM runtime.
2. Full instance-to-instance reachability (T3) — acceptable single-user; P1 SG rules.
3. Docker Desktop quirks (rootless options, firewall bypass, netem absence) — documented per-OS caveats.
4. Disk-fill within a volume until P1 per-volume enforcement.
5. Local token can be read by any process running as the same user — out of threat model (same user already owns the machine).
