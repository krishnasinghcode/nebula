#!/bin/sh
# Build the curated Orbit images (see orbit/images/).
set -eu
cd "$(dirname "$0")/../orbit/images"
for img in ubuntu-24.04 debian-12; do
  docker build -t "nebula/orbit-${img}:latest" "${img}"
done
