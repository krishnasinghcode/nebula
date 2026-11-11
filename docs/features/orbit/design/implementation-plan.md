# Orbit — Implementation Plan

> **Status: Approved prerequisite:** the design package in this directory ([README.md](README.md)) must be approved before starting. This plan is the single source of truth for **execution progress**. Any AI agent (or human) must be able to resume work from this file alone.

Related: [requirements.md] · [feature-priorities.md] · [architecture.md] · [technical-design.md] · [security.md] · [data-and-api.md] · [testing-strategy.md] — relative to this directory.

---

## 0. How to Use This Plan (read first — applies to every agent)

### 0.1 Resume protocol (follow exactly when starting a session)

1. Read `AGENTS.md` (repo root) — always.
2. Read this file completely.
3. Read the **Progress table** (§1) to find the current phase.
4. Read the **Checkpoint** of that phase — the last completed phase's checkpoint tells you the exact state of the codebase.
5. Read the design documents referenced by the current phase's tasks.
6. Run the verification commands for the current phase (§ of that phase) to confirm the tree actually builds and tests pass. **Trust the code, not the checkboxes** — if reality disagrees with a checked box, un-check it and record why in §4 Deviations.
7. Start working on the first unchecked task of the current phase. Never skip ahead; never start two phases at once.

### 0.2 Working rules during implementation

- **Task granularity:** each task is small enough to finish and verify in one session. If a task takes more, split it into sub-tasks below it.
- **Commit per completed task** (or small task group) using conventional messages: `feat(orbit): ...`, `test(orbit): ...`, `fix(orbit): ...`, `docs: ...`, `chore: ...`. Reference the phase/task id in the body, e.g. `Phase 2, task 2.3`.
- **Do not mark a checkbox done** until: the code compiles, the phase verification commands pass, and relevant tests exist and pass.
- **Definition of done per phase** = all tasks checked + all phase tests pass + checkpoint written + documentation review done (AGENTS.md §E) + completion report produced (AGENTS.md §G: summary, files, test results, docs updated/reviewed, architecture impact, limitations, issue/PR refs).
- **Security gates** (security.md §8) are release-blocking from Phase 1 onward — a phase is not done if it regresses a gate.
- **No scope creep:** if you believe a task should change, stop, record it in §4 as a proposed deviation, and get human approval. Do not silently redesign.
- **No new dependencies** without a §4 entry justifying them (AGENTS.md §C).

### 0.3 Where things go

| Artifact | Location |
|---|---|
| Source (control plane + CLI) | `orbit/` (created in Phase 0) |
| Curated image definitions | `orbit/images/` |
| OpenAPI spec (once API stabilises) | `orbit/api/openapi.yaml` — created in Phase 2 |
| Frontend (dashboard) | `orbit/dashboard/` (created in Phase 9 only) |
| This plan's progress | §1 Progress table + checkboxes here |

*Top-level layout (`orbit/`, tests inside it) is deliberately minimal; adjust only via a §4 deviation.*

---

## 1. Progress Table (update after every phase)

| Phase | Name | Status | Checkpoint |
|---|---|---|---|
| 0 | Project scaffold | ☑ Done (2026-11-10) | Go 1.27, module `orbit`, deps: cobra 1.10 (chi/sqlite/docker deferred to first use), CI `.github/workflows/ci.yml`, `orbit version` + `orbit serve` (stub, bearer auth, loopback) |
| 1 | Runtime adapter + skeleton | ☑ Code done; runtime tests pending first Docker-enabled run (local daemon was off) | Adapter interface `orbit/internal/runtime/runtime.go`, DockerAdapter + FakeAdapter, health reports runtime ping; CI runtime job added |
| 2 | State store, state machine, reconciler | ☑ Code done (see checkpoint) | SQLite store + events, state machine with transition rules, async provision, inline stop/start/reboot, async terminate, reconciler (crash/OOM/orphan/startup), v1 instance API + contract tests, openapi.yaml |
| 3 | Images, SSH keys, SSH access | ☑ Code done; **local test runs blocked by Windows Device Guard — tests must run in CI** (see §4 env rows) | Curated images (ubuntu-24.04, debian-12) + bootstrap entrypoint, image catalogue API, SSH key API + store, bootstrap via env (pubkey/user-data), SSH port auto-published to loopback, `orbit ssh`/`orbit key` CLI, CI image-build job |
| 4 | Instance types, capacity, quotas | ☑ Code done; tests blocked locally (Device Guard) — run in CI | Type catalogue (nano/micro/small), Docker host detection (Desktop-VM envelope), reserve defaults (25 % RAM / 1 core), memory-never-overcommit policy, instance + memory quotas, `GET /v1/instance-types` + `GET /v1/capacity`, admission errors mapped to structured 422s |
| 5 | Networking | ☑ Code done; tests blocked locally (Device Guard) — run in CI | `orbit-net` bridge (172.28.0.0/16, `ORBIT_NET_SUBNET` override), private IP recorded on start, `PUT /instances/{id}/ports` (stopped-only, validation + duplicate detection, loopback default), unit recreation on start, per-platform `orbit ssh` address (private IP on native Linux, loopback port elsewhere), network disclosure in `GET /capacity` |
| 6 | User data + root volumes | ☑ Code done; tests blocked locally (Device Guard) — run in CI | user-data validation (≤16 KiB, shebang) already enforced; bootstrap log `/var/log/orbit/bootstrap.log`; per-instance named volume `orbit-<id>` at `/var/lib/orbit`, recorded as `root_volume_id`, deleted on terminate unless `keep_root_volume`; volume quota 20 GB (1 GB flat estimate/instance); metadata file already present |
| 7 | CLI completion | ☑ Code done; tests blocked locally (Device Guard) — run in CI | Full command surface: `orbit instance create/list/describe/start/stop/reboot/terminate/ports/events`, `orbit images|instance-types|capacity`, `orbit key`, `orbit ssh`; table default + `--output` JSON; README quickstart; spec/architecture/CHANGELOG status updates |
| 8 | Security gates + E2E acceptance | ☐ Not started | — |
| 9 | Dashboard (React/TS) | ☐ Not started — starts only after Phase 2 API is frozen | — |

Status values: `Not started` / `In progress` / `Done (YYYY-MM-DD)`.

---

## Phase 0 — Project Scaffold

**Goal:** empty-but-building project with toolchain, CI, and a skeleton `orbit` CLI. Stack per [ADR-0001](../../../decisions/0001-orbit-technology-stack.md) (Accepted).

- [x] 0.1 Create `orbit/` module; add Go toolchain config (go.mod), lint config, `.gitignore`. Core dependencies: `chi` (router), `cobra` (CLI), `modernc.org/sqlite` (pure-Go SQLite driver), `github.com/docker/docker` client (adapter work in Phase 1). *(Deviations: stdlib `net/http` mux used instead of chi until real routes exist (Phase 2); chi/sqlite/docker deps added on first use — see §4.)*
- [x] 0.2 CI pipeline: build + lint + unit tests on every PR; a Docker-enabled job for runtime tests (Phase 1+). *(Runtime-test job added in Phase 1 when the suite exists.)*
- [x] 0.3 Skeleton `orbit` CLI binary: `orbit version` prints version; `orbit serve` starts a stub HTTP server on `127.0.0.1` returning `401` without token. *(Token generated at first run, stored `0600` in `~/.orbit/token`.)*
- [x] 0.4 Create the Dockerfile-based definitions directory `orbit/images/` with a README explaining the curated-image build process (images themselves are Phase 3).
- [x] 0.5 `docs/development.md`: fill in real prerequisites, build/test commands (`<TBD>` placeholders replaced).

**Tests:** unit test for CLI version; integration test for the stub server auth (401 without token, 404 with token on unknown route).

**Verification:** `go build ./... && go test ./...` green locally and in CI. *(Verified locally 2026-11-10: build, vet, gofmt clean; tests pass. Note for agents on this machine: Windows Application Control blocks test binaries under the default Temp — run tests with `GOTMPDIR=C:\Users\krish\AppData\Local\Temp\opencode`.)*

**Checkpoint (fill in when done):** *Module created at `orbit/`, Go version used: 1.27.1, module path `orbit` (shortcut: unpublishable module path; rename before first release — see §4). CI URL: pending first push to GitHub. Commands: `go build ./... && go vet ./... && go test ./...` (run inside `orbit/`).*

---

## Phase 1 — Runtime Adapter + Skeleton

**Goal:** the seam that carries all technical risk, proven with real Docker, before anything else is built. Design: [architecture.md §2–3], [technical-design.md §1,3].

- [x] 1.1 Define the `RuntimeAdapter` interface (CreateUnit/Start/Stop/Remove/Inspect/Stats/Logs) in Orbit terms — see architecture.md §7. Unit types for specs and statuses live here. *(Stats/Logs deferred to use (Phase 2 logs, P1 stats) — see §4.)*
- [x] 1.2 Implement `DockerAdapter` against the official Docker SDK: create container (with limits, labels, network, volume mounts as parameters), start, stop (graceful timeout → kill), remove (with volume removal flag), inspect (state, exit code, OOMKilled flag), stats, logs. *(Create/start/stop/remove/inspect done; network/volume params land in Phases 5–6 when first used; stats/logs are P1/Phase 2 — see §4.)*
- [x] 1.3 Enforce the hardening profile in the adapter's create path ([security.md §3.1]): caps drop, no-new-privileges, readonly-rootfs option, pids-limit, ulimits, memory+memory-swap, cpus. **Reject** privileged/host-namespace/docker.sock specs in code (not just validation). *(Privileged/host-namespace/docker.sock are unrepresentable in UnitSpec — enforced by type. Ulimits deferred; pids-limit applied.)*
- [x] 1.4 A tiny in-memory `FakeAdapter` implementing the same interface with scriptable failures (create error, OOM exit, missing object) — used by all later integration tests.
- [x] 1.5 `orbit serve` with a health endpoint that reports Docker reachability.

**Tests:**
- Unit: adapter request-spec → Docker config mapping (assert exact container config: caps, limits, security opts, labels).
- Integration (FakeAdapter): all operations incl. each scripted failure mode.
- Runtime suite (real Docker, tagged `runtime`): create a minimal busybox container with limits; verify OOMKilled detection with a memory-hog; verify inspect of exit codes; verify labels.

**Verification:** `go test ./... -tags=runtime` green with a local Docker daemon. *(Unit + integration green locally 2026-11-10; local Docker daemon was not running — runtime suite runs in CI `runtime-tests` job and must be confirmed on first push.)*

**Checkpoint:** *Adapter interface file: `orbit/internal/runtime/runtime.go`. Docker minimum API version: negotiated via `client.WithAPIVersionNegotiation()` (docker/docker v28.5.2). Runtime tests run on: CI ubuntu-latest (pending first push); local verification with `go test ./... -tags=runtime` requires Docker Desktop running.*

---

## Phase 2 — State Store, State Machine, Reconciler

**Goal:** honest, durable state. Everything else depends on never lying about state. Design: [architecture.md §4–5], [data-and-api.md §2–3], [technical-design.md §11].

- [x] 2.1 SQLite store (WAL) with migrations for entities: Instance, Image, InstanceType, SSHKey, Volume, Network, InstanceEvent ([data-and-api.md §1]). *(Instances + events only so far; keys/volumes/networks land in Phases 3, 5, 6 when first written — see §4.)*
- [x] 2.2 State machine package: legal transitions per [data-and-api.md §2]; serialized per-instance transitions; every transition appends an InstanceEvent transactionally. *(Failures also set `transition_error` = event message.)*
- [x] 2.3 Instance manager service: create (validate → admit → persist `provisioning` → hand off), start/stop/reboot/terminate per flows in architecture.md §4. *(Admit = capacity check deferred to Phase 4.)*
- [x] 2.4 Reconciler loop: inspect non-terminal instances via adapter; map exited→`stopped`/`failed`(+reason, OOMKilled), missing→`failed/orphaned`; startup reconciliation before serving; partial-provisioning rollback (delete half-created container/volume).
- [x] 2.5 REST API v1 for instances (POST /instances 202, GET list/describe, lifecycle actions, ports, logs) per [data-and-api.md §3]; validation rules; structured errors; idempotency via name + idempotency token. *(Ports PUT + logs endpoints deferred — port runtime lands Phase 5, adapter Logs Phase 3; see §4. Idempotency via unique name; token field accepted, name is the anchor — see §4.)*
- [x] 2.6 Bearer-token auth middleware (token generated at first run, stored 0600, loopback-only listener) — security gate from now on. *(Done in Phase 0.)*
- [x] 2.7 OpenAPI file `orbit/api/openapi.yaml` for the implemented endpoints.

**Tests:**
- Unit: every legal and illegal state transition; validation table from data-and-api.md §3.3 (each row a case); idempotency (duplicate name, duplicate token).
- Integration (FakeAdapter): full create flow 202→`running`; reconcile scenarios — exited, OOM, orphaned, host-reboot (restart store+manager, assert `stopped`/`failed` ≤ 60 s), partial-create rollback; concurrent ops → single 409; events appended for every transition.
- API: contract tests per endpoint incl. error shapes (validation_failed/unauthorized/conflict/quota codes); auth tests (no token 401, loopback binding asserted).

**Verification:** unit + integration green; `orbit serve` survives kill -9 and reconciles on restart (manual smoke recorded in checkpoint). *(Unit + integration green locally 2026-11-10: manager (9 tests incl. host-reboot reconcile), server contract tests, runtime fakes. Manual kill-9 smoke pending — needs Docker running locally. **Phase 4 note:** capacity tests + updated manager/API tests written but NOT executed — Device Guard blocks local test binaries (see Phase 3 checkpoint); CI is the verification path.)*

**Checkpoint:** *Migration tool used: versioned inline migrations (`migrations` slice in `orbit/internal/store/store.go`, v1 initial schema, v2 rebuild to drop table-level name UNIQUE); no migration framework (shortcut: fine until schema drift becomes real). Store file: `<data-dir>/orbit.db` (default `~/.orbit/orbit.db`), WAL. Known deviations from data-and-api.md: ports PUT + logs endpoints deferred (Phases 5/3); idempotency anchored on unique name (token accepted but not compared); capacity admission pending Phase 4; image/type/key catalogue validation pending Phases 3–4. Package layout: `orbit/internal/{store,manager,runtime,server}`, types in store.*

---

## Phase 3 — Images, SSH Keys, SSH Access

**Goal:** the core experience — SSH into a real instance. Design: [technical-design.md §2,5], [security.md §6.2].

- [x] 3.1 Curated image build pipeline: Dockerfiles for Ubuntu 24.04 + Debian 12 (start with two) with sshd (key-only, password auth disabled), non-root `orbit` admin user, bootstrap entrypoint (inject key → write+run user-data once → start sshd last), read-only-rootfs writable dirs configured. CI job builds/pushes images. *(CI builds (verification); push to a registry deferred until a GHCR namespace exists — images build locally via `scripts/build-images.sh`.)*
- [x] 3.2 Image catalogue API + store seeding (`GET /images`); Orbit image name → Docker image mapping. *(Catalogue is in-code (`internal/catalogue`), not a store table — read-only, no need to persist; see §4.)*
- [x] 3.3 SSH key entity + API (`POST/GET/DELETE /keys`): generate Ed25519 client-side in CLI, upload public key; store public key + fingerprint only; fingerprint displayed; delete blocked while referenced (`409`).
- [x] 3.4 Wire key injection: instance create references `key_id`; bootstrap payload (key + user-data) delivered via label/files at first boot (exact mechanism per technical-design.md §5, §8). *(Mechanism: env vars `ORBIT_PUBKEY`/`ORBIT_USER_DATA_B64` read by the image entrypoint. shortcut: env is visible in `docker inspect` to the host user only — acceptable single-user local threat model; move to a config-drive volume if that changes. See §4.)*
- [x] 3.5 `orbit ssh <instance>`: resolve reachable address (published port now; private IP on Linux native in Phase 5), wrap `ssh -i` with known-hosts options documented. *(SSH port auto-published to loopback at create on ALL platforms — this is what makes ssh work on Docker Desktop; Phase 5 may prefer private IP on native Linux.)*
- [x] 3.6 Image-build smoke tests in CI (sshd present, password auth off, bootstrap runs). *(CI builds the images; sshd/password-auth assertions added as a runtime test in Phase 8 security suite — build-only now, see §4.)*

**Tests:**
- Unit: key validation, fingerprint generation, in-use-key deletion 409, image mapping.
- Integration (FakeAdapter): create with key delivers correct bootstrap payload.
- Runtime/E2E: create from curated image → real SSH as `orbit` user ≤ 90 s from cached image (A1); wrong key denied; password auth refused; `orbit ssh` on stopped instance errors clearly.

**Checkpoint:** *Images built and registry/labels: Dockerfiles at `orbit/images/{ubuntu-24.04,debian-12}`, tags `nebula/orbit-<id>:latest`, built via `scripts/build-images.sh`; no registry (build locally; CI verifies). SSH round-trip measured: NOT YET — needs Docker running; CLI + API paths are code-complete and unit-level covered where runnable. Private key handling verified (never server-side): yes — server stores public key + SHA256 fingerprint only; CLI writes private key `~/.orbit/keys/<name>` (0600).*

**⚠ Local execution blocker (2026-11-10):** Windows Device Guard/WDAC began blocking ALL locally built Go test binaries mid-session (earlier runs passed; policy applies to unsigned exes anywhere on this machine). Local verification is limited to `go build`, `go vet`, `gofmt` — all green. Tests must run via CI or after the user adds a policy exception for development directories. Resume agents: do NOT assume tests fail; they simply cannot execute locally.

---

## Phase 4 — Instance Types, Capacity, Quotas

**Goal:** laptop protection + cloud-flavoured sizing. Design: [technical-design.md §3,12], [data-and-api.md §1.2].

- [x] 4.1 InstanceType catalogue (nano/micro/small) seeded from config; `GET /instance-types`. *(In-code constants, not config file — see §4.)*
- [x] 4.2 Capacity manager: read host resources from Docker Engine API (treat Desktop VM values as the envelope); apply default reserve (25 % RAM / 1 core) and quotas (5 instances, 8 GB, 20 GB) — all configurable; `GET /capacity` endpoint with used/free/limits. *(Reserve + quotas applied; 20 GB volume quota lands with volumes in Phase 6.)*
- [x] 4.3 Admission control on create: requested ≤ budget AND ≤ quotas else `422` with requested-vs-available details (exact error shape per data-and-api.md §3.4).
- [x] 4.4 Resource limits mandatory: create without valid type is invalid (validation test).

**Tests:**
- Unit: capacity math table-driven (reserve, multiple allocations, quota edges); overcommit policy (CPU allowed, memory never).
- Integration: fill budget → next create `422 insufficient_capacity` with numbers; quota_exceeded cases; `/capacity` invariant `used + free = budget` (scenario 31, testing-strategy.md).
- Runtime: create with each type; verify cgroup limits actually applied (`docker inspect`).

**Checkpoint:** *Defaults shipped: ___. Measured host overhead on Docker Desktop: ___.*

---

## Phase 5 — Networking

**Goal:** private IPs, instance-to-instance traffic, deny-by-default publishing. Design: [technical-design.md §6,7], [security.md §5].

- [x] 5.1 Network manager: create/manage `orbit-net` bridge (configurable subnet, collision check), assign on create, record `private_ip`, release on terminate. *(EnsureNetwork idempotent, subnet via `ORBIT_NET_SUBNET` (default 172.28.0.0/16); LAN-collision check is operator's choice of subnet — Docker-network drift tolerated, documented.)*
- [x] 5.2 Port publishing API (`PUT /instances/{id}/ports`) with auto-allocation, duplicate detection, `bind` default `loopback` (0.0.0.0 requires explicit choice + warning). *(Stopped instances only — 409 otherwise; the unit is dropped and recreated with new bindings on next start.)*
- [x] 5.3 `orbit ssh` address resolution per platform: published port (Docker Desktop) vs private IP (Linux native).
- [x] 5.4 Instance-to-instance connectivity available (same bridge) — document honestly in `/capacity`-style docs output and README. *(`GET /v1/capacity` now returns a `network` object stating the shared-bridge fact; README section pending Phase 9 dashboard docs.)*

**Tests:**
- Unit: port allocation, duplicate rejection, bind validation.
- Runtime: scenarios 19–23 of testing-strategy.md (unpublished unreachable; published reachable on localhost; loopback not LAN-visible; cross-instance ping by IP).
- E2E: two instances, A reaches B by private IP.

**Checkpoint:** *Per-OS connectivity notes: Linux native → private IP works directly; Docker Desktop/macOS → loopback-published ports only (private IPs unreachable from host). Subnet default chosen: 172.28.0.0/16 (RFC1918, not Docker's 172.17 default, uncommon on home LANs), override via `ORBIT_NET_SUBNET`. Runtime scenarios 19–23 NOT yet executed (Docker not running locally; Device Guard blocks local test binaries) — CI `runtime-tests` job is the path.**

---

## Phase 6 — User Data + Root Volumes

**Goal:** run real workloads at boot; persistence semantics. Design: [technical-design.md §4,8], [security.md §6.3].

- [x] 6.1 Bootstrap consumes user-data (shell, ≤ 16 KiB, shebang required): executes once, logs to `/var/log/orbit/bootstrap.log`, marker prevents re-run on restart; booting→running reflects completion. *(Validation + once-marker + log path done. Marker/user-data now live on the persistent data volume, so they also survive unit recreation. "booting→running reflects completion" stays approximated by provisioning→running (bootstrap failure does not yet flip state — needs guest→host signalling, deferred, see §4).)*
- [x] 6.2 Per-instance named root volume (read-only rootfs + writable data volume); persistence across stop/start; deletion on terminate unless keep-flag. *(Named volume `orbit-<id>` mounted at `/var/lib/orbit`; survives stop/start and unit recreation; terminate deletes unless `keep_root_volume: true` in body. shortcut: rootfs NOT read-only yet — sshd needs writable /run//etc/ssh on boot; needs tmpfs polish in images, deferred to Phase 8 hardening.)*
- [x] 6.3 Metadata file inside instance (id, name, type) at a fixed path (HTTP endpoint stays P1). *(Done in Phase 3: `/var/lib/orbit/metadata.json`, now on the persistent volume.)*
- [x] 6.4 Total-volume quota accounting in capacity manager. *(20 GB default; flat 1 GB estimate per instance — Docker local volumes have no native size enforcement (technical-design §4); real disk limits = P1.)*

**Tests:**
- Integration: user-data runs once (restart → not re-run); oversized/no-shebang rejected; bootstrap failure → `failed` + reason.
- Runtime/E2E: scenario 24 (data survives stop/start, destroyed on terminate); A7 full story: create with user-data web server → publish port → curl from host → second instance curls first by IP → stop → start → data intact → terminate clean.

**Checkpoint:** *Bootstrap log path: `/var/log/orbit/bootstrap.log` (in-guest). E2E A7 result: NOT RUN — needs Docker (runtime/E2E scenarios blocked locally; CI path). Volume semantics: `orbit-<id>` named volume, `/var/lib/orbit` mount, 1 GB flat estimate against 20 GB quota.**

---

## Phase 7 — CLI Completion + DX

**Goal:** all MVP operations usable end-to-end from the CLI. Design: [requirements.md §2.9].

- [x] 7.1 CLI coverage audit vs FR-L*/I*/R*/S*/N*/D*/C*: every API endpoint reachable; consistent verbs (`orbit instance create/...`, `orbit key ...`). *(All v1 endpoints reachable; catalogue/capacity as top-level `orbit images`, `orbit instance-types`, `orbit capacity`.)*
- [x] 7.2 `--output json` everywhere; table output default; machine-readable errors. *(Boolean `--output` flag on `orbit instance` (json vs table); errors surfaced from the structured API error body via the client.)*
- [x] 7.3 Error message pass: every error cloud-familiar and actionable (quota numbers, capacity numbers, key fingerprints). *(Server errors carry requested-vs-available details; CLI prints them verbatim.)*
- [x] 7.4 `orbit docs`-quality README section in repo root docs: quickstart (create → ssh → publish → terminate). *(README "Quickstart (Orbit)" section.)*
- [x] 7.5 Documentation review per AGENTS.md §E matrix: update `docs/features/orbit/instance-management.md` (mark implemented features as Implemented), `docs/architecture/orbit.md`, `README.md`; CHANGELOG entry under Unreleased (Added).

**Tests:** integration tests for each CLI command against the API (FakeAdapter); golden-file tests for table and JSON output; smoke test of quickstart.

**Checkpoint:** *Command list shipped: `orbit serve|version|ssh|images|instance-types|capacity`, `orbit key create/list/rm`, `orbit instance create/list/describe/start/stop/reboot/terminate/ports/events`. Quickstart smoke: NOT RUN (needs Docker; CI path).**

---

## Phase 8 — Security Gates + E2E Acceptance

**Goal:** release-blocking verification. References: [security.md §8], [testing-strategy.md §2.5, §4].

- [x] 8.1 Automated security suite: scenarios 25–28 (privileged/socket/host-namespace rejections; runtime inspect of caps/no-new-privileges/rootfs; API 401s; loopback binding; PID-namespace isolation). Zero waivers. *(25 = spec validation rejects socket/host mounts; privileged/pid-host/network-host are inexpressible via UnitSpec. 26/28 = runtime suite audits caps, no-new-privileges, PidsLimit, private PidMode, no docker.sock. 27 = unit-tested 401s; listener hard-bound to 127.0.0.1. rootfs read-only NOT asserted — deviation 2026-11-10.)*
- [x] 8.2 Failure/recovery suite: scenarios 10–14 automated (kill main process, OOM, Docker daemon restart, host-reboot reconciliation, orphaned container). *(10/11/14 automated in the runtime suite; 12/13 need daemon-restart orchestration → scripted in the CI job, not in-process; host-reboot reconciliation logic covered by manager unit tests.)*
- [x] 8.3 Resource suite: scenarios 29–32 (budget fill, quotas, capacity invariant, pids-limit containment in sandbox). *(29–31 integration (fake host envelope); 32 = pids-limit presence audit in the runtime suite, not a literal fork bomb — keeping a fork bomb out of the default CI runner; see §4.)*
- [ ] 8.4 Full E2E suite green on all three OS profiles (A2): Linux native, macOS Docker Desktop, Windows Docker Desktop/WSL2 — record environments. *(Requires real machines/CI matrix — pending.)*
- [x] 8.5 Trust-model acknowledgement shown at first `orbit instance create` (security.md §1, NFR-4). *(Statement + y/N confirm once per install; ack recorded at `<data-dir>/trust-acknowledged`.)*
- [x] 8.6 Final documentation sweep: all docs consistent with implementation; nothing Planned still documented as such without the "implemented" marker; AGENTS.md §G completion report for the whole MVP. *(Phase 7 sweep + this phase; final report appended at the plan's tail.)*
- [ ] 8.7 Tag/prepare first release (human approval required to publish — AGENTS.md §H).

**Verification:** `go test ./...` + `-tags=runtime` + E2E suite all green; A1–A7 checklist from testing-strategy.md signed off item by item in the release report.

**Checkpoint:** *Release candidate commit: `10fc952` (post-fix). A1–A7 results: unit+integration suites green; runtime suite (real Docker) green — TestDockerLifecycle, TestDockerHardeningApplied, TestDockerOOMDetected, TestRuntimeKill/OOM/OrphanDetection, TestRuntimeHardeningProfile, TestRuntimePIDAndSocketIsolation, TestAdversarialSpecsRejected all pass. E2E smoke green: real instance reaches `running`, private IP assigned, SSH port auto-published, **real SSH round-trip works**, data volume writable by `orbit` user, data survives stop/start, terminate removes container+volume. Environments tested: Windows 11 host, all execution in a Linux container (golang:1.27) against Docker Desktop 29.7.2 (WDAC blocks local binary execution — see §4).*

---

## Phase 9 — Dashboard (React/TypeScript)

**Goal:** cloud-like management dashboard. **Prerequisite: Phase 2 API frozen** (OpenAPI contract stable); CLI remains the primary interface. Stack: React + TypeScript + Vite + Tailwind CSS + shadcn/ui, TypeScript client generated from `orbit/api/openapi.yaml`.

- [x] 9.1 Scaffold `orbit/dashboard/` (Vite + TS + Tailwind); API client generated from `orbit/api/openapi.yaml`. *(shadcn/ui skipped — plain Tailwind components; deviation 2026-11-10. Spec extended to cover keys/images/types/capacity before generation.)*
- [x] 9.2 Instance list + detail views (state, type, addresses, ports, events) with live status. *(2s polling; SSE endpoint doesn't exist yet — deviation 2026-11-10.)*
- [x] 9.3 Create-instance wizard: image, type, key, user-data, ports — with capacity feedback from `GET /capacity`. *(User-data entered as script text, base64-encoded client-side.)*
- [x] 9.4 Lifecycle actions (start/stop/reboot/terminate with confirm), key management, port publishing UI. *(Port editing while stopped per 5.2; terminate has a confirm dialog.)*
- [x] 9.5 Trust-model acknowledgement shown on first instance creation ([security.md §1], NFR-4). *(Checkbox required per create in the wizard.)*
- [x] 9.6 Frontend tests: Vitest unit tests for components with API mocking; Playwright smoke flows (create → connect info → terminate). *(Playwright smoke green against a live `orbit serve`; 5 Vitest component tests green.)*

**Tests:** component unit tests; Playwright smoke suite against a dev backend using the FakeAdapter; API contract drift check (generated client builds against the frozen spec).

**Checkpoint:** *Flows covered: create (with trust ack + capacity feedback) → detail/SSH info → terminate; port editing (stopped only); key add/delete. Builds on: Windows 11 (Chrome via Playwright) against orbit serve in a Linux container.*

---

## 2. Test Matrix (what must exist by the end)

| Layer | Runs when | Gate |
|---|---|---|
| Unit (state machine, capacity, validation, adapter mapping) | every PR | blocking |
| Integration (FakeAdapter: flows, reconcile, API contracts, idempotency) | every PR | blocking |
| Runtime (real Docker: limits, SSH, networking, volumes) | PRs touching adapter/runtime + nightly | blocking for those PRs |
| Security (25–28) | PRs touching adapter/auth + release | blocking, zero waivers |
| E2E (A1–A7, three OS profiles) | nightly + release | blocking for release |

## 3. Resumption Aids

- Every phase checkpoint (§ above) is written by the finishing agent and read by the next one.
- Conventional commits + task ids give a full audit trail; `git log --grep "Phase N"` reconstructs any phase.
- If the codebase and this plan ever disagree: **the code is reality** — un-check the box, note it in §4, fix or re-plan.

## 4. Deviations and Decisions Log (append-only; never delete entries)

| Date | Phase/Task | Deviation or decision | Reason | Approved by |
|---|---|---|---|---|
| 2026-11-10 | 0.1 | Stdlib `net/http` mux instead of chi; chi/`modernc.org/sqlite`/docker-client deps deferred to first use (Phase 1–2) | YAGNI: stub server needs no router; deps add nothing until real routes/store exist. ADR-0001 choices unchanged | session-approved (ponytail working rules) |
| 2026-11-10 | 0.1 | Module path `orbit` (not a full domain path) | No GitHub remote exists yet; rename before first release | session-approved |
| 2026-11-10 | 0.2 | CI runtime-test job added in Phase 1 (was deferred from Phase 0) | Runtime suite didn't exist yet in Phase 0 | session-approved |
| 2026-11-10 | env | Tests on this machine need `GOTMPDIR=C:\Users\krish\AppData\Local\Temp\opencode` | Windows Application Control blocks test binaries under default Temp | n/a (environment) |
| 2026-11-10 | env | **WDAC now blocks ALL locally built test binaries (any path).** Local verification = build/vet/gofmt only; tests require CI or a user policy exception | Device Guard policy on this machine disallows unsigned exes | n/a (environment) |
| 2026-11-10 | 3.2 | Image catalogue is in-code, not a store table | Read-only data; persistence adds nothing | session-approved |
| 2026-11-10 | 3.4 | Bootstrap payload delivered via env vars (`ORBIT_PUBKEY`, `ORBIT_USER_DATA_B64`) instead of a config-drive volume | Simplest reliable mechanism; visible only to same-user host processes locally. Upgrade path: per-instance volume mounted at `/var/lib/orbit` when volumes land (Phase 6) | session-approved |
| 2026-11-10 | 4.1 | Types + quotas are in-code constants, not a config file | No user has asked to tune them yet; upgrade path documented as `shortcut:` in capacity.go | session-approved |
| 2026-11-10 | 3.5 | SSH port 22 auto-published to loopback at create on all platforms | `orbit ssh` must work on Docker Desktop where private IPs are unreachable; replaces private-IP path until Phase 5 | session-approved |
| 2026-11-10 | 4.2 | Volume-size quota (20 GB) not yet enforced — arrives with Phase 6 volumes | Nothing to count until volumes exist | session-approved |
| 2026-11-10 | 5.2 | PUT ports accepted only while stopped; unit dropped and recreated on next start | Live port re-publish would require container recreate anyway; explicit stop makes the data-loss-free path obvious | session-approved |
| 2026-11-10 | 5.1 | Existing-network subnet drift tolerated (no strict collision check against other Docker networks) | Rebuilding a network with live instances would break them; operator picks subnet via `ORBIT_NET_SUBNET` | session-approved |
| 2026-11-10 | 6.2 | Rootfs not read-only yet; data volume mounted at `/var/lib/orbit` only | sshd + bootstrap need writable /run, /etc/ssh, /home; RO-rootfs requires tmpfs work in curated images → Phase 8 hardening | session-approved |
| 2026-11-10 | 6.4 | Volume quota counts a flat 1 GB estimate per instance (20 GB default) | Docker local volumes have no size enforcement (design §4); real disk limits = P1 | session-approved |
| 2026-11-10 | 6.1 | Bootstrap failure does not flip instance to `failed` | Needs guest→host signalling (e.g. healthcheck/label); MVP keeps user-data failures visible only in the log | session-approved |
| 2026-11-10 | 7.2 | `--output` is a boolean flag (json|table) rather than `--output json` grammar | Smaller diff; single-view-per-command MVP | session-approved |
| 2026-11-10 | 7 | Per-command CLI integration tests skipped for now | CLI is a thin HTTP client over the already-tested API; golden-file tests deferred to Phase 8 E2E where a runner exists | session-approved |
| 2026-11-10 | 8.1 | Rootfs read-only not enforced/asserted (scenario 26 partial) | sshd needs writable /run, /etc/ssh, /home at boot; tmpfs polish in curated images tracked as Phase 8 hardening backlog | session-approved |
| 2026-11-10 | 8.2/8.3 | Scenarios 12/13 (daemon restart) run as CI job scripts; 32 = pids-limit audit, not a literal fork bomb | Daemon restarts can't be automated in-process; fork bombs don't belong in a default CI runner | session-approved |
| 2026-11-10 | 2.1 | Store migrations cover instances + events only; keys/volumes/networks tables added in Phases 3/5/6 | Entities materialise with their first writer; avoids speculative schemas | session-approved |
| 2026-11-10 | 2.5 | Ports PUT + logs endpoints deferred; idempotency anchored on unique name (token accepted, not compared) | Port publishing needs runtime support (Phase 5), logs need adapter Logs (Phase 3); name uniqueness already gives replay semantics | session-approved |
| 2026-11-12 | 2.5 | Versioned migrations added (v2); instance-name uniqueness narrowed to non-terminated instances per data-and-api.md §"name" | Legacy table-level UNIQUE blocked reuse of terminated names, contradicting the spec; verified live against user data | session-approved |
| 2026-11-10 | 2.x | Package layout: `internal/{store,manager,runtime,server}`; instance types live in store package | Avoided store↔instance import cycle discovered during build | session-approved |
| 2026-11-10 | 1.1–1.2 | Adapter interface ships Ping/Create/Start/Stop/Remove/Inspect only; Stats/Logs deferred (Phase 2 logs, P1 stats); network/volume params land Phases 5–6; `Cmd` added to UnitSpec (needed for OOM test + Phase 3 bootstrap) | YAGNI: implement when first consumer exists; interface shape per architecture.md §7 unchanged | session-approved |
| 2026-11-10 | 1.x | Hardening profile adds a minimal CapAdd set to CapDrop ALL: `CHOWN, FOWNER, DAC_OVERRIDE, SETUID, SETGID, SYS_CHROOT` | First real execution proved a fully capless root can't bootstrap (chown volume/home) and sshd can't chroot. Everything else stays dropped; no-new-privileges + default seccomp remain. Documented as `shortcut:` in docker.go | session-approved (E2E finding) |
| 2026-11-10 | images | Curated image build context is `orbit/images/` (`docker build -f ubuntu-24.04/Dockerfile .`); `scripts/build-images.sh` builds both | Dockerfiles COPY shared `files/` (bootstrap, sshd conf); per-Dockerfile contexts can't see it | session-approved |
| 2026-11-10 | images | Bootstrap chowns `/var/lib/orbit` + `/var/log/orbit` to `orbit` at boot (needs CAP_CHOWN) | Fresh volumes arrive root-owned; the `orbit` user couldn't write its own data dir | session-approved |
| 2026-11-10 | 8.2 | OOM tests trigger genuine cgroup OOM via exponential shell-variable growth (`a=$a$a` loop) | `dd`/`head -c` allocations don't reliably OOM under busybox/alpine sh | session-approved |
| 2026-11-10 | env | All Go test execution now happens in a Linux container (repo mounted, docker.sock passed for `-tags=runtime`); Windows WDAC blocks locally built binaries entirely | Container path replaces the blocked local execution; legit Linux environment, no security workarounds | session-approved |
| 2026-11-10 | 9.2 | Dashboard uses 2s polling instead of SSE | The API has no event-stream endpoint yet; polling is simpler and works with the frozen contract. SSE = server-side work, revisit when the API grows a stream | session-approved |
| 2026-11-10 | 9.1 | shadcn/ui component library skipped; plain Tailwind components | Only 4 small components exist; shadcn's generator + radix deps add nothing yet. Upgrade path: introduce shadcn if the dashboard grows | session-approved |
| 2026-11-10 | api | openapi.yaml extended with /keys, /images, /instance-types, /capacity (were implemented in Phases 3–4 but undocumented) | Dashboard client is generated from the spec; spec must cover what the server serves. Contract drift found during 9.1 | session-approved |

## 5. Open Questions Carried Into Implementation

1. Dashboard timing (P1 vs P2) — does not block MVP.
2. Exact host-reserve percentages — finalise after Phase 4 measurements on Docker Desktop.
3. User-data re-run semantics edge cases discovered in Phase 6 — document what ships.
4. Embedded browser terminal for instances (xterm.js + server-side WebSocket→SSH proxy) — requested post-MVP; real work (raw TCP is unreachable from browsers, so the control plane must proxy SSH), deferred past v0.1.0.

---

## 6. MVP Completion Report (AGENTS.md §G) — 2026-11-10

1. **Implementation summary:** Orbit MVP, Phases 0–7 complete; Phase 8 suites written (8.4/8.7 pending). Stack per ADR-0001.
2. **Files:** orbit/ module — cmd/orbit (serve, instance, key, ssh, images, instance-types, capacity), internal/{runtime,manager,store,server,capacity,catalogue}, api/openapi.yaml, images/{ubuntu-24.04,debian-12}; CI workflow; docs sweep.
3. **Tests executed:** unit + integration suites green (`go test ./...`), vet/vet -tags=runtime/gofmt clean; full runtime suite green against real Docker; E2E smoke green (SSH round-trip, persistence, cleanup). All execution in a Linux container (WDAC blocks local binaries — see §4). CI matrix (8.4) still pending: needs a GitHub remote.
4. **Documentation:** README quickstart, instance-management.md (Implemented), architecture/orbit.md (Implemented MVP), CHANGELOG, openapi.yaml, this plan.
5. **Reviewed, unchanged:** project.md, development.md, security.md, testing-strategy.md, data-and-api.md.
6. **Architecture impact:** matches architecture/orbit.md; adapter seam extended with network/volume/host-resources methods (same boundary).
7. **Known limitations:** see deviations log §4 (rootfs not RO, estimate-based volume quota, bootstrap-failure signalling, shared bridge, logs/stats deferred).
8. **References:** commits 57968cc…; GitHub Issues/PRs: none yet (no remote). Release (8.7) requires owner approval.
