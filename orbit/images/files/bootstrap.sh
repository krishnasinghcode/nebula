#!/bin/sh
# Orbit instance bootstrap (runs as root at container start).
# Inputs via env (delivered by the control plane):
#   ORBIT_PUBKEY     — user's SSH public key (required)
#   ORBIT_USER_DATA_B64 — base64-encoded startup script (optional)
#   ORBIT_INSTANCE_META — JSON instance metadata (id, name, type)
set -eu

install -d -m 700 -o orbit -g orbit /home/orbit/.ssh
printf '%s\n' "$ORBIT_PUBKEY" > /home/orbit/.ssh/authorized_keys
chown orbit:orbit /home/orbit/.ssh/authorized_keys
chmod 600 /home/orbit/.ssh/authorized_keys

printf '%s\n' "${ORBIT_INSTANCE_META:-{}}" > /var/lib/orbit/metadata.json

# User data: write once, run once as the orbit admin user.
if [ -n "${ORBIT_USER_DATA_B64:-}" ] && [ ! -f /var/lib/orbit/user-data-ran ]; then
  printf '%s' "$ORBIT_USER_DATA_B64" | base64 -d > /var/lib/orbit/user-data
  chmod +x /var/lib/orbit/user-data
  if su orbit -c "/var/lib/orbit/user-data" > /var/log/orbit-bootstrap.log 2>&1; then
    touch /var/lib/orbit/user-data-ran
  else
    echo "user-data script failed; see /var/log/orbit-bootstrap.log" >&2
  fi
fi

exec /usr/sbin/sshd -D -e
