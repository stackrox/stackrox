# Interim devcontainer image based on UBI9, replacing the apollo-ci image
# until it is updated to UBI9 upstream.
# Replicates the essential toolchain from rox-ci-image (stackrox-build + stackrox-test).

FROM registry.access.redhat.com/ubi9:latest

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# ---------- Base packages (from stackrox-build) ----------
# Enable EPEL for packages not in UBI9 base repos (e.g. tmux, needed by paude wrapper).
RUN dnf update -y && \
    dnf install -y \
        dnf-plugins-core \
        wget \
        && \
    dnf config-manager --set-enabled ubi-9-codeready-builder-rpms && \
    dnf update -y && \
    dnf install -y \
        autoconf \
        automake \
        binutils \
        gcc \
        gcc-c++ \
        gdb \
        glibc-devel \
        libtool \
        make \
        pkgconf \
        pkgconf-pkg-config \
        redhat-rpm-config \
        strace \
        git \
        git-core \
        source-highlight \
        bzip2-devel \
        gettext \
        jq \
        zstd \
        lz4-devel \
        procps-ng \
        zlib-devel \
        unzip \
        xz \
        zip \
        lsof \
        openssl \
        socat \
        glibc-langpack-en \
        diffutils \
	vim \
        && \
    dnf clean all && \
    rm -rf /var/cache/dnf /var/cache/yum

# ---------- tmux (not in UBI9 repos, install from CentOS Stream 9) ----------
RUN printf '[centos-stream-9-baseos]\nname=CentOS Stream 9 - BaseOS\nbaseurl=https://mirror.stream.centos.org/9-stream/BaseOS/$basearch/os/\ngpgcheck=0\nenabled=0\n' \
        > /etc/yum.repos.d/centos-stream9-baseos.repo && \
    dnf --enablerepo=centos-stream-9-baseos install -y tmux && \
    dnf clean all && \
    rm -rf /var/cache/dnf /var/cache/yum

# ---------- Node.js + Yarn ----------
RUN dnf module enable -y nodejs:20 && \
    dnf install -y nodejs && \
    npm install -g yarn && \
    dnf clean all && \
    rm -rf /var/cache/dnf /var/cache/yum

# ---------- Go ----------
ARG GOLANG_VERSION=1.25.7
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH
RUN ARCH="$(uname -m)" && \
    case "$ARCH" in x86_64) GOARCH=amd64;; aarch64) GOARCH=arm64;; *) echo "unsupported arch: $ARCH" >&2; exit 1;; esac && \
    wget --no-verbose -O go.tgz "https://dl.google.com/go/go${GOLANG_VERSION}.linux-${GOARCH}.tar.gz" && \
    tar -C /usr/local -xzf go.tgz && \
    rm go.tgz && \
    mkdir -p "$GOPATH/src" "$GOPATH/bin" && \
    chmod -R 777 "$GOPATH"

ENV CGO_ENABLED=1

# ---------- PostgreSQL client (from stackrox-test) ----------
RUN ARCH="$(uname -m)" && \
    dnf install -y "https://download.postgresql.org/pub/repos/yum/reporpms/EL-9-${ARCH}/pgdg-redhat-repo-latest.noarch.rpm" && \
    dnf --disablerepo="*" --enablerepo="pgdg15" install -y \
        postgresql15 \
        postgresql15-server \
        && \
    dnf clean all && \
    rm -rf /var/cache/dnf /var/cache/yum

# ---------- Docker CLI (from stackrox-test) ----------
ARG DOCKER_VERSION=29.2.1
RUN ARCH="$(uname -m)" && \
    wget --no-verbose -O /tmp/docker.tgz \
        "https://download.docker.com/linux/static/stable/${ARCH}/docker-${DOCKER_VERSION}.tgz" && \
    tar -xz -C /tmp -f /tmp/docker.tgz && \
    install /tmp/docker/docker /usr/local/bin && \
    rm -rf /tmp/docker /tmp/docker.tgz

# ---------- Helm (from stackrox-test) ----------
RUN ARCH="$(uname -m)" && \
    case "$ARCH" in x86_64) HELMARCH=amd64;; aarch64) HELMARCH=arm64;; *) echo "unsupported arch: $ARCH" >&2; exit 1;; esac && \
    wget --no-verbose -O helm.tgz "https://get.helm.sh/helm-v3.11.2-linux-${HELMARCH}.tar.gz" && \
    tar -xf helm.tgz && \
    install "linux-${HELMARCH}/helm" /usr/local/bin && \
    rm -rf helm.tgz "linux-${HELMARCH}"

# ---------- Shellcheck (from stackrox-test) ----------
ARG SHELLCHECK_VERSION=0.10.0
RUN ARCH="$(uname -m)" && \
    wget --quiet "https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/shellcheck-v${SHELLCHECK_VERSION}.linux.${ARCH}.tar.xz" && \
    tar -xJf "shellcheck-v${SHELLCHECK_VERSION}.linux.${ARCH}.tar.xz" && \
    install "shellcheck-v${SHELLCHECK_VERSION}/shellcheck" /usr/local/bin && \
    rm -rf "shellcheck-v${SHELLCHECK_VERSION}.linux.${ARCH}.tar.xz" "shellcheck-v${SHELLCHECK_VERSION}"

# ---------- Bats (from stackrox-test) ----------
RUN npm install -g bats@1.10.0 bats-support@0.3.0 bats-assert@2.0.0 tap-junit

# ---------- Ralph Orchestrator ----------
ARG RALPH_VERSION=2.8.1
RUN ARCH="$(uname -m)" && \
    case "$ARCH" in x86_64) TRIPLE=x86_64-unknown-linux-gnu;; aarch64) TRIPLE=aarch64-unknown-linux-gnu;; *) echo "unsupported arch: $ARCH" >&2; exit 1;; esac && \
    wget --no-verbose -O ralph-cli.tar.xz "https://github.com/mikeyobrien/ralph-orchestrator/releases/download/v${RALPH_VERSION}/ralph-cli-${TRIPLE}.tar.xz" && \
    tar -xJf ralph-cli.tar.xz && \
    install ralph-cli-${TRIPLE}/ralph /usr/local/bin && \
    rm -rf ralph-cli.tar.xz ralph-cli-${TRIPLE}

# ---------- yq (from stackrox-test) ----------
RUN ARCH="$(uname -m)" && \
    case "$ARCH" in x86_64) YQARCH=amd64;; aarch64) YQARCH=arm64;; *) echo "unsupported arch: $ARCH" >&2; exit 1;; esac && \
    wget --no-verbose "https://github.com/mikefarah/yq/releases/download/v4.16.2/yq_linux_${YQARCH}" && \
    install "yq_linux_${YQARCH}" /usr/local/bin/yq && \
    rm "yq_linux_${YQARCH}"

WORKDIR /go/src/github.com/stackrox/rox
