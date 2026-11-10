# Orbit Images (curated)

Dockerfiles for the images users launch Orbit Instances from (Phase 3 of the
implementation plan). Each image ships an SSH server configured for key-only
auth, a non-root `orbit` admin user, and the bootstrap entrypoint (inject key →
run user-data once → start sshd last). See
`docs/features/orbit/design/technical-design.md` §2 and §8.

Planned images: `debian-12`, `ubuntu-24.04` (more later: alpine, fedora).

Built by CI; users never build these themselves.
