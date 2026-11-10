# Nebula Orbit curated image: Ubuntu 24.04
# Built as nebula/orbit-ubuntu-24.04:latest (scripts/build-images.sh)
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
      openssh-server sudo ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY files/bootstrap.sh /usr/local/sbin/orbit-bootstrap
COPY files/sshd-orbit.conf /etc/ssh/sshd_config.d/orbit.conf
# Build from the images/ directory: docker build -f ubuntu-24.04/Dockerfile .
RUN chmod +x /usr/local/sbin/orbit-bootstrap \
    && useradd -m -s /bin/bash -G sudo orbit \
    && echo 'orbit ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/orbit \
    && mkdir -p /var/run/sshd /var/lib/orbit /var/log/orbit

EXPOSE 22
# sshd stays in the foreground; bootstrap runs as the orbit user's setup step.
CMD ["/usr/local/sbin/orbit-bootstrap"]
