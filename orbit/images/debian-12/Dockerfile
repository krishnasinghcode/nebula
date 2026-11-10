# Nebula Orbit curated image: Debian 12
# Built as nebula/orbit-debian-12:latest (scripts/build-images.sh)
FROM debian:12

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
      openssh-server sudo ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY files/bootstrap.sh /usr/local/sbin/orbit-bootstrap
COPY files/sshd-orbit.conf /etc/ssh/sshd_config.d/orbit.conf
RUN chmod +x /usr/local/sbin/orbit-bootstrap \
    && useradd -m -s /bin/bash -G sudo orbit \
    && echo 'orbit ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/orbit \
    && mkdir -p /var/run/sshd /var/lib/orbit /var/log/orbit

EXPOSE 22
CMD ["/usr/local/sbin/orbit-bootstrap"]
