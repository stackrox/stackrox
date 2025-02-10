FROM quay.io/fedora/fedora:latest

RUN mkdir -p /stackrox/static-data && dnf install -y postgresql elfutils-libelf libbpf
COPY image/rhel/static-bin/* /usr/bin
RUN save-dir-contents /etc/pki/ca-trust /etc/ssl

COPY bundle/nvd_definitions /nvd_definitions
COPY bundle/k8s_definitions /k8s_definitions
COPY bundle/istio_definitions /istio_definitions
COPY bundle/repo2cpe /repo2cpe
COPY scannerv2/image/scanner/dump/genesis_manifests.json /
COPY bundle/genesis-dump.zip /

COPY data /stackrox-data
COPY image/rhel/docs /stackrox/static-data/docs
COPY bin/* /stackrox
RUN mkdir -p /stackrox/bin && \
    ln -s /stackrox/migrator /stackrox/bin/migrator && \
    ln -s /stackrox/self-checks /usr/local/bin/self-checks
COPY ui/build /ui
