ARG MAPPINGS_REGISTRY=registry.access.redhat.com
ARG MAPPINGS_BASE_IMAGE=ubi8
ARG MAPPINGS_BASE_TAG=latest
ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-minimal
ARG BASE_TAG=latest

FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_1.20 as builder

ENV CGO_ENABLED=1
ENV GOFLAGS=""
ENV GOTAGS="release"
ENV RELEASE_TAG="$CI_VERSION"
ENV CI=1

COPY . /src
WORKDIR /src

RUN make -C scanner NODEPS=1 CGO_ENABLED=${CGO_ENABLED} image/scanner/bin/scanner copy-scripts

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

LABEL \
    com.redhat.component="rhacs-scanner-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="This image supports image scanning for RHACS" \
    io.k8s.description="This image supports image scanning for RHACS" \
    io.k8s.display-name="scanner" \
    io.openshift.tags="rhacs,scanner,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-scanner-rhel8" \
    source-location="https://github.com/stackrox/scanner" \
    summary="The image scanner for RHACS" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    # TODO(ROX-20236): configure injection of dynamic version value when it becomes possible.
    version="0.0.1-todo"

COPY --from=builder \
    /src/scanner/image/scanner/scripts/entrypoint.sh \
    /src/scanner/image/scanner/scripts/import-additional-cas \
    /src/scanner/image/scanner/scripts/restore-all-dir-contents \
    /src/scanner/image/scanner/scripts/save-dir-contents \
    /src/scanner/image/scanner/bin/scanner \
    /usr/local/bin/

# COPY --from=builder /src/scanner/image/scanner/THIRD_PARTY_NOTICES/ /THIRD_PARTY_NOTICES/

# The mapping files are not optional.
# The helm chart hard codes in the indexer config the path to the mapping
# files, which is preferred over downloading.  If the file does not exist, the
# indexer raises an error (i.e. it does not fall back to download).

COPY ./repository-to-cpe.json ./container-name-repos-map.json /run/mappings/

RUN microdnf upgrade --nobest && \
    microdnf clean all && \
    # (Optional) Remove line below to keep package management utilities
    rpm -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum && \
    chown -R 65534:65534 /tmp && \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 65534:65534 /etc/pki /etc/ssl && save-dir-contents /etc/pki/ca-trust /etc/ssl

# This is equivalent to nobody:nobody.
USER 65534:65534

ENTRYPOINT ["entrypoint.sh"]
