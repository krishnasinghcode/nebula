# Orbit Usage Guide

End-to-end guide: start Orbit, create and access an instance, manage it, and clean up. Everything runs on your machine with Docker.

**What you'll use:**

| Piece | What it is | Where it runs |
|---|---|---|
| Control plane (`orbit serve`) | The "brain": API, database, reconciler | Terminal / container, port 7420 |
| Dashboard | Web UI for everything below | Your browser, port 5173 |
| Orbit instances | Real Ubuntu/Debian containers managed for you | Created automatically, don't touch them directly |

---

## 1. Start the backend

Requirements: [Docker](https://docs.docker.com/get-docker/) running.

```sh
cd orbit
go build -o orbit ./cmd/orbit
./orbit serve
```

You'll see:

```
orbit control plane listening on http://127.0.0.1:7420 (data: ~/.orbit)
```

Leave this terminal open. The API listens on loopback only — nothing is exposed to your network.

> **Can't run the binary locally?** (e.g. Windows execution policy) run the control plane in a container instead. The `docker rm -f` line makes this command **idempotent** — run it again any time and it replaces the old container; the `orbit-data` volume keeps your keys and token across recreations:
> ```sh
> docker rm -f orbit 2>/dev/null
> docker run -d --name orbit -p 127.0.0.1:7420:7420 -e ORBIT_ADDR=0.0.0.0:7420 \
>   -v "$PWD:/src" -v /var/run/docker.sock:/var/run/docker.sock \
>   -v orbit-data:/root/.orbit \
>   -w /src -e GOCACHE=/tmp/gocache -e DOCKER_HOST=unix:///var/run/docker.sock \
>   golang:1.27 sh -c "go build -o /bin/orbit ./cmd/orbit && /bin/orbit serve"
> ```
> All `orbit` commands below then run as `docker exec orbit /bin/orbit ...`.

## 2. Get your API token

The first `orbit serve` start generated a token — a random secret that authenticates every API call (including the dashboard).

```sh
cat ~/.orbit/token
```

(In the container setup: `docker exec orbit cat /root/.orbit/token`.)

## 3. Start the dashboard (optional but recommended)

In a **second terminal**:

```sh
cd orbit/dashboard
npm install        # first time only
npm run dev
```

Open **http://127.0.0.1:5173**, paste the token from step 2, click Connect. It's remembered in your browser.

Everything below can be done from the dashboard UI or the CLI — both talk to the same API.

## 4. Register an SSH key

This is how you log in to instances. Orbit only stores your **public** key.

Dashboard: bottom panel "SSH keys" → name it `dev`, paste the contents of your public key (`~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`) → **Add key**.

No key yet? Generate one: `ssh-keygen -t ed25519`.

CLI equivalent:

```sh
orbit key create dev --public-key "$(cat ~/.ssh/id_ed25519.pub)"
orbit key list
```

## 5. Create an instance

Dashboard: "Create instance" form → pick a **name** (lowercase, e.g. `web-1`), an **image** (Ubuntu 24.04), a **type** (e.g. *1 vCPU / 1 GB*), your key → tick the trust-model acknowledgement → **Create**.

> **About the trust-model acknowledgement:** Orbit instances are containers sharing your host kernel. They are not a security boundary between users — the acknowledgement makes sure you know this. It appears once per create, by design (see [security model](design/security.md)).

The instance starts in **provisioning** and reaches **running** within seconds. The capacity line under the form shows how much of your machine is already allocated.

CLI equivalent:

```sh
orbit instance create web-1 --image ubuntu-24.04 --type orbit.micro --key dev
orbit instance list                 # watch state → running
```

Optional flags:
- `--port 8080:80` — publish the instance's port 80 on your machine's port 8080 (loopback)
- `--user-data "$(cat setup.sh)"` — a shell script that runs on first boot

## 6. Access and use the instance

**SSH in:**

```sh
orbit ssh web-1
```

No key juggling — Orbit finds the key and the auto-published SSH port for you. Inside, it's a real Ubuntu machine: try `cat /var/lib/orbit/metadata.json`, install packages, run a server.

**From your browser/terminal on the host:** anything you published with `--port` is on `http://127.0.0.1:<host-port>` once the service inside is running.

Dashboard: click the instance row → detail view shows the SSH command, private IP, published ports, and lifecycle buttons.

## 7. Manage it

All available from the dashboard detail view or the CLI:

```sh
orbit instance stop web-1      # graceful stop; ports stay configured
orbit instance start web-1     # boot again — data survives
orbit instance reboot web-1    # running only
orbit instance describe web-1  # full state, IPs, ports, event history
orbit capacity                 # host envelope, usage, quotas
```

Notes:
- **Data persists across stop/start.** Each instance gets a private volume mounted at `/var/lib/orbit` — files you write there survive restarts. Everything outside it (installed packages except /home) resets on stop/start like an ephemeral OS disk.
- **Port changes apply while stopped.** Stop → edit ports in the dashboard (or `PUT /v1/instances/{id}/ports`) → start.
- **Names are unique.** Creating with an existing name of a live instance returns the existing one (idempotent).

## 8. Terminate and clean up

```sh
orbit instance terminate web-1
```

Dashboard: detail view → **Terminate** → confirm.

This is irreversible: the container is removed and the data volume is deleted (that's the EC2 behavior too). The record stays visible as `terminated` for auditability.

## Troubleshooting

| Symptom | Fix |
|---|---|
| Dashboard says 401 / "unauthorized" | Token wrong or backend restarted with a new one — re-run step 2 |
| `connection refused` on 7420/5173 | Backend or dashboard terminal isn't running (steps 1/3) |
| Instance stuck in `failed` | `orbit instance describe <name>` shows the transition error; common cause: image not built — see [images README](../orbit/images/README.md) |
| SSH: connection closed / permission denied | The key registered in Orbit must match a private key on your machine (`~/.ssh/`) |
| `insufficient capacity` on create | You hit a quota — check `orbit capacity`, terminate something, or use a smaller type |

## Where to go next

- Architecture and internals: [docs/architecture/orbit.md](../architecture/orbit.md)
- Feature specification: [docs/features/orbit/instance-management.md](instance-management.md)
- Security model and trust boundaries: [docs/features/orbit/design/security.md](design/security.md)
- API reference: [orbit/api/openapi.yaml](../../orbit/api/openapi.yaml)
