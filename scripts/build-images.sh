#!/bin/sh
# Build the curated Orbit images (see orbit/images/).
# Context is orbit/images so both Dockerfiles can COPY files/.
set -eu
cd "$(dirname "$0")/../orbit/images"
for img in ubuntu-24.04 debian-12; do
  docker build -f "${img}/Dockerfile" -t "nebula/orbit-${img}:latest" .
done
