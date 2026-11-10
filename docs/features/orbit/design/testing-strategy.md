# Orbit — Testing Strategy

> **Status: Proposed.** Verification approach for the MVP defined in [feature-priorities.md](feature-priorities.md). Measurable acceptance criteria are stated per area; the MVP "definition of done" is at the end.

## 1. Test Layers

| Layer | Scope | Runtime profile |
|---|---|---|
| **Unit** | State machine, capacity math, validation, idempotency logic, API handlers (mocked adapter) | Fast (< 1 min), no Docker, every commit |
| **Integration** | Control plane + real SQLite + a **fake runtime adapter** (scriptable failures: create error, OOM, missing container) | Fast, every commit |
| **Runtime (Docker)** | Real Docker Engine: container/volume/network ops, limits actually enforced | Requires Docker; CI job + local `make test-runtime` |
| **End-to-end (E2E)** | Full CLI → API → Docker → SSH flows on a real host | Slow; nightly + release gate |
| **Security** | Hardening-profile enforcement, adversarial create specs, socket/privilege rejections | Security gate per PR touching runtime/adapter |

## 2. Scenario Catalogue

### 2.1 Instance lifecycle (P0)

1. Create valid instance → reaches `running` ≤ 60 s from cached image (NFR-3).
2. Create with unknown image/type/key → `422`, nothing provisioned (no orphan container/volume — verify via Docker).
3. Duplicate create (same idempotency token, then same name) → same instance returned, exactly one container exists.
4. Stop → `stopped` ≤ 30 s; volume data persists; start → files intact (write marker file, stop, start, assert).
5. Reboot → `running`, uptime reset, data intact.
6. Terminate → container and root volume removed (verify via Docker), record `terminated`, events logged.
7. Terminate while `provisioning` → no leaks; final state `terminated`.
8. Concurrent ops (stop + terminate) → one `409 conflict`, state consistent.
9. Every transition appends an `InstanceEvent`; `describe` shows history.

### 2.2 Runtime & reconciliation

10. `kill -9` the guest's main process (or `docker kill`) → reconciler sets `failed` with reason ≤ 30 s, never stale `running`.
11. OOM simulation (guest program allocates beyond limit) → state `failed`, reason `OOMKilled`, host healthy.
12. Stop Docker daemon mid-operation → API returns `503 runtime_error`; retry after daemon restart succeeds; no half-state.
13. Host-reboot simulation: restart Docker + Orbit with instances "running" → reconciliation to `stopped`/`failed`, no stale state, `GET /instances` honest within 60 s of start.
14. Orphaned container (deleted out-of-band) → reconciler marks `failed/orphaned`, does not crash.

### 2.3 SSH

15. Create with key → bootstrap injects authorized_keys; `ssh` (non-interactive) succeeds ≤ 90 s from cached image.
16. Password auth attempt → refused (image config test).
17. Wrong key → permission denied (expected).
18. `orbit ssh` on a stopped instance → clear error, not a hang.

### 2.4 Networking & storage

19. Fresh instance: no published ports → instance port **unreachable** from host (connection refused/timeout).
20. Publish 8080 (auto host port) → reachable at `localhost:<port>`; second publish gets a different free port.
21. Publish with `bind=loopback` → **not** reachable from another machine on the LAN; `bind=all` reachable (explicit).
22. Two instances: A reaches B by private IP (ICMP/TCP) — the distributed-workload core case.
23. Subnet collision config (user sets a LAN-overlapping subnet) → either clear startup error or working isolation; no silent breakage.
24. Data written to a known path survives stop/start; destroyed after terminate (per FR-D1).

### 2.5 Security (gate per §8 of security.md)

25. Create spec with `Privileged=true` / docker.sock mount / `PidMode=host` / `NetworkMode=host` → rejected `422`; container never created.
26. Running container profile audit: caps = expected minimum, `no-new-privileges` effective, rootfs read-only (assert via inspect in the runtime suite).
27. API without token → `401`; token from another install → `401`; listener bound to loopback only.
28. Instance process cannot see host processes (PID namespace) nor reach the Docker socket (blocked/no socket present in guest).

### 2.6 Resource & capacity

29. Fill the schedulable budget with instances → next create `422 insufficient_capacity` with requested-vs-available numbers.
30. Quota breach (max instances / total memory) → `422 quota_exceeded`.
31. `GET /capacity` matches `sum(instance allocations) + free = budget` invariant.
32. pids-limit: guest fork bomb is contained (host responsive; guest hits limit) — run only in the sandboxed runtime suite.

## 3. Reliability Requirements

- Unit + integration suites green on every PR; runtime suite green on every PR touching the adapter/runtime; E2E green nightly and as a release gate.
- No test may depend on external internet beyond first-time image pulls (CI pre-pulls curated images).
- Flaky E2E tests are treated as bugs: retried once, then triaged with an issue.
- Reconciliation invariants hold at all times: no non-terminal instance without a live runtime object *and* no state that lies about runtime reality for > 60 s.

## 4. MVP Acceptance Criteria (measurable)

| # | Criterion |
|---|---|
| A1 | Create-from-cached-image → `running` ≤ 60 s; SSH-ready ≤ 90 s (typical laptop: 4 cores/16 GB). |
| A2 | Scenario 4 (stop/start persistence) passes on all three OS profiles (Linux native, macOS DD, Windows DD/WSL2). |
| A3 | Scenarios 25–28 (security gates) pass with zero waivers. |
| A4 | Host never consumes > 50 % RAM under default quotas with a full default-quota instance set running. |
| A5 | Reboot-host scenario (13) reconciles honestly ≤ 60 s after control-plane start. |
| A6 | All MVP API endpoints behave per [data-and-api.md] contracts, including error shapes and idempotency. |
| A7 | The full MVP story (create → user-data app → publish port → curl from host → cross-instance ping → stop → data intact → terminate clean) passes as one E2E test. |
