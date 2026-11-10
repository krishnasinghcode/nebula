#!/bin/sh
# E2E SSH check from inside the orbit-e2e container via the Docker gateway.
GW=$(getent hosts host.docker.internal | awk '{print $1}')
[ -n "$GW" ] || GW=172.17.0.1
ssh -i /root/.orbit/keys/dev -o IdentitiesOnly=yes \
    -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -p "$1" orbit@"$GW" "${2:-echo SSH-OK}"
