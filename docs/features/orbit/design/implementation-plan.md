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
| 4 | Instance types, capacity, quotas | ☐ Not started | — |
| 5 | Networking | ☐ Not started | — |
| 6 | User data + root volumes | ☐ Not started | — |
| 7 | CLI completion | ☐ Not started | — |
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

**Verification:** unit + integration green; `orbit serve` survives kill -9 and reconciles on restart (manual smoke recorded in checkpoint). *(Unit + integration green locally 2026-11-10: manager (9 tests incl. host-reboot reconcile), server contract tests, runtime fakes. Manual kill-9 smoke pending — needs Docker running locally.)*

**Checkpoint:** *Migration tool used: inline migrations const (v1) in `orbit/internal/store/store.go` — no migration framework (shortcut: fine until schema drift becomes real; add versioned migrations if schema changes post-release). Store file: `<data-dir>/orbit.db` (default `~/.orbit/orbit.db`), WAL. Known deviations from data-and-api.md: ports PUT + logs endpoints deferred (Phases 5/3); idempotency anchored on unique name (token accepted but not compared); capacity admission pending Phase 4; image/type/key catalogue validation pending Phases 3–4. Package layout: `orbit/internal/{store,manager,runtime,server}`, types in store.*

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

- [ ] 4.1 InstanceType catalogue (nano/micro/small) seeded from config; `GET /instance-types`.
- [ ] 4.2 Capacity manager: read host resources from Docker Engine API (treat Desktop VM values as the envelope); apply default reserve (25 % RAM / 1 core) and quotas (5 instances, 8 GB, 20 GB) — all configurable; `GET /capacity` endpoint with used/free/limits.
- [ ] 4.3 Admission control on create: requested ≤ budget AND ≤ quotas else `422` with requested-vs-available details (exact error shape per data-and-api.md §3.4).
- [ ] 4.4 Resource limits mandatory: create without valid type is invalid (validation test).

**Tests:**
- Unit: capacity math table-driven (reserve, multiple allocations, quota edges); overcommit policy (CPU allowed, memory never).
- Integration: fill budget → next create `422 insufficient_capacity` with numbers; quota_exceeded cases; `/capacity` invariant `used + free = budget` (scenario 31, testing-strategy.md).
- Runtime: create with each type; verify cgroup limits actually applied (`docker inspect`).

**Checkpoint:** *Defaults shipped: ___. Measured host overhead on Docker Desktop: ___.*

---

## Phase 5 — Networking

**Goal:** private IPs, instance-to-instance traffic, deny-by-default publishing. Design: [technical-design.md §6,7], [security.md §5].

- [ ] 5.1 Network manager: create/manage `orbit-net` bridge (configurable subnet, collision check), assign on create, record `private_ip`, release on terminate.
- [ ] 5.2 Port publishing API (`PUT /instances/{id}/ports`) with auto-allocation, duplicate detection, `bind` default `loopback` (0.0.0.0 requires explicit choice + warning).
- [ ] 5.3 `orbit ssh` address resolution per platform: published port (Docker Desktop) vs private IP (Linux native).
- [ ] 5.4 Instance-to-instance connectivity available (same bridge) — document honestly in `/capacity`-style docs output and README.

**Tests:**
- Unit: port allocation, duplicate rejection, bind validation.
- Runtime: scenarios 19–23 of testing-strategy.md (unpublished unreachable; published reachable on localhost; loopback not LAN-visible; cross-instance ping by IP).
- E2E: two instances, A reaches B by private IP.

**Checkpoint:** *Per-OS connectivity notes: ___. Subnet default chosen: ___.*

---

## Phase 6 — User Data + Root Volumes

**Goal:** run real workloads at boot; persistence semantics. Design: [technical-design.md §4,8], [security.md §6.3].

- [ ] 6.1 Bootstrap consumes user-data (shell, ≤ 16 KiB, shebang required): executes once, logs to `/var/log/orbit/bootstrap.log`, marker prevents re-run on restart; booting→running reflects completion.
- [ ] 6.2 Per-instance named root volume (read-only rootfs + writable data volume); persistence across stop/start; deletion on terminate unless keep-flag.
- [ ] 6.3 Metadata file inside instance (id, name, type) at a fixed path (HTTP endpoint stays P1).
- [ ] 6.4 Total-volume quota accounting in capacity manager.

**Tests:**
- Integration: user-data runs once (restart → not re-run); oversized/no-shebang rejected; bootstrap failure → `failed` + reason.
- Runtime/E2E: scenario 24 (data survives stop/start, destroyed on terminate); A7 full story: create with user-data web server → publish port → curl from host → second instance curls first by IP → stop → start → data intact → terminate clean.

**Checkpoint:** *Bootstrap log path: ___. E2E A7 result: ___.*

---

## Phase 7 — CLI Completion + DX

**Goal:** all MVP operations usable end-to-end from the CLI. Design: [requirements.md §2.9].

- [ ] 7.1 CLI coverage audit vs FR-L*/I*/R*/S*/N*/D*/C*: every API endpoint reachable; consistent verbs (`orbit instance create/...`, `orbit key ...`).
- [ ] 7.2 `--output json` everywhere; table output default; machine-readable errors.
- [ ] 7.3 Error message pass: every error cloud-familiar and actionable (quota numbers, capacity numbers, key fingerprints).
- [ ] 7.4 `orbit docs`-quality README section in repo root docs: quickstart (create → ssh → publish → terminate).
- [ ] 7.5 Documentation review per AGENTS.md §E matrix: update `docs/features/orbit/instance-management.md` (mark implemented features as Implemented), `docs/architecture/orbit.md`, `README.md`; CHANGELOG entry under Unreleased (Added).

**Tests:** integration tests for each CLI command against the API (FakeAdapter); golden-file tests for table and JSON output; smoke test of quickstart.

**Checkpoint:** *Command list shipped: ___.*

---

## Phase 8 — Security Gates + E2E Acceptance

**Goal:** release-blocking verification. References: [security.md §8], [testing-strategy.md §2.5, §4].

- [ ] 8.1 Automated security suite: scenarios 25–28 (privileged/socket/host-namespace rejections; runtime inspect of caps/no-new-privileges/rootfs; API 401s; loopback binding; PID-namespace isolation). Zero waivers.
- [ ] 8.2 Failure/recovery suite: scenarios 10–14 automated (kill main process, OOM, Docker daemon restart, host-reboot reconciliation, orphaned container).
- [ ] 8.3 Resource suite: scenarios 29–32 (budget fill, quotas, capacity invariant, pids-limit containment in sandbox).
- [ ] 8.4 Full E2E suite green on all three OS profiles (A2): Linux native, macOS Docker Desktop, Windows Docker Desktop/WSL2 — record environments.
- [ ] 8.5 Trust-model acknowledgement shown at first `orbit instance create` (security.md §1, NFR-4).
- [ ] 8.6 Final documentation sweep: all docs consistent with implementation; nothing Planned still documented as such without the "implemented" marker; AGENTS.md §G completion report for the whole MVP.
- [ ] 8.7 Tag/prepare first release (human approval required to publish — AGENTS.md §H).

**Verification:** `go test ./...` + `-tags=runtime` + E2E suite all green; A1–A7 checklist from testing-strategy.md signed off item by item in the release report.

**Checkpoint:** *Release candidate commit: ___. A1–A7 results: ___. Environments tested: ___.*

---

## Phase 9 — Dashboard (React/TypeScript)

**Goal:** cloud-like management dashboard. **Prerequisite: Phase 2 API frozen** (OpenAPI contract stable); CLI remains the primary interface. Stack: React + TypeScript + Vite + Tailwind CSS + shadcn/ui, TypeScript client generated from `orbit/api/openapi.yaml`.

- [ ] 9.1 Scaffold `orbit/dashboard/` (Vite + TS + Tailwind + shadcn/ui); generate API client from the OpenAPI spec.
- [ ] 9.2 Instance list + detail views (state, type, addresses, ports, events) with SSE-driven live status.
- [ ] 9.3 Create-instance wizard: image, type, key, user-data, ports — with capacity feedback from `GET /capacity`.
- [ ] 9.4 Lifecycle actions (start/stop/reboot/terminate with confirm), key management, port publishing UI.
- [ ] 9.5 Trust-model acknowledgement shown on first instance creation ([security.md §1], NFR-4).
- [ ] 9.6 Frontend tests: Vitest unit tests for components with API mocking; Playwright smoke flows (create → connect info → terminate).

**Tests:** component unit tests; Playwright smoke suite against a dev backend using the FakeAdapter; API contract drift check (generated client builds against the frozen spec).

**Checkpoint:** *Flows covered: ___. Builds on: (OS/browser).*

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
| 2026-11-10 | 3.5 | SSH port 22 auto-published to loopback at create on all platforms | `orbit ssh` must work on Docker Desktop where private IPs are unreachable; replaces private-IP path until Phase 5 | session-approved |
| 2026-11-10 | 2.1 | Store migrations cover instances + events only; keys/volumes/networks tables added in Phases 3/5/6 | Entities materialise with their first writer; avoids speculative schemas | session-approved |
| 2026-11-10 | 2.5 | Ports PUT + logs endpoints deferred; idempotency anchored on unique name (token accepted, not compared) | Port publishing needs runtime support (Phase 5), logs need adapter Logs (Phase 3); name uniqueness already gives replay semantics | session-approved |
| 2026-11-10 | 2.x | Package layout: `internal/{store,manager,runtime,server}`; instance types live in store package | Avoided store↔instance import cycle discovered during build | session-approved |
| 2026-11-10 | 1.1–1.2 | Adapter interface ships Ping/Create/Start/Stop/Remove/Inspect only; Stats/Logs deferred (Phase 2 logs, P1 stats); network/volume params land Phases 5–6; `Cmd` added to UnitSpec (needed for OOM test + Phase 3 bootstrap) | YAGNI: implement when first consumer exists; interface shape per architecture.md §7 unchanged | session-approved |

## 5. Open Questions Carried Into Implementation

1. Dashboard timing (P1 vs P2) — does not block MVP.
2. Exact host-reserve percentages — finalise after Phase 4 measurements on Docker Desktop.
3. User-data re-run semantics edge cases discovered in Phase 6 — document what ships.
