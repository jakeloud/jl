FROM debian:11-slim

RUN apt-get update && apt-get install -y \
    systemd \
    nginx \
    certbot \
    python3-certbot-nginx \
    docker.io \
    openssh-client \
    git \
    && rm -rf /var/lib/apt/lists/*
