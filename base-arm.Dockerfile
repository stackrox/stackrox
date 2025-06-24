FROM quay.io/fedora/fedora:latest

RUN dnf install -y postgresql elfutils-libelf libbpf nodejs npm
RUN curl -L "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl" > /usr/bin/kubectl && \
    chmod +x /usr/bin/kubectl
