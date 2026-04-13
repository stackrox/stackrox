FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25@sha256:bd531796aacb86e4f97443797262680fbf36ca048717c00b6f4248465e1a7c0c AS builder

ARG BUILD_TAG
RUN if [[ "$BUILD_TAG" == "" ]]; then >&2 echo "error: required BUILD_TAG arg is unset"; exit 6; fi
ENV BUILD_TAG="$BUILD_TAG"

ENV GOFLAGS=""
# TODO(ROX-20240): enable non-release development builds.
# TODO(ROX-27054): Remove the redundant strictfipsruntime option if one is found to be so.
ENV GOTAGS="release,strictfipsruntime"
ENV GOEXPERIMENT=strictfipsruntime
ENV CI=1

COPY . /src
WORKDIR /src

RUN make -C scanner NODEPS=1 CGO_ENABLED=1 image/scanner/bin/scanner copy-scripts

FROM registry.access.redhat.com/ubi9/ubi-micro:latest@sha256:093a704be0eaef9bb52d9bc0219c67ee9db13c2e797da400ddb5d5ae6849fa10 AS ubi-micro-base

FROM registry.access.redhat.com/ubi9/ubi:latest@sha256:6ed9f6f637fe731d93ec60c065dbced79273f1e0b5f512951f2c0b0baedb16ad AS package_installer

COPY --from=ubi-micro-base / /out/

RUN dnf install -y \
    --installroot=/out/ \
    --releasever=9 \
    --setopt=install_weak_deps=0 \
    --setopt=reposdir=/etc/yum.repos.d \
    --nodocs \
    ca-certificates openssl && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

FROM ubi-micro-base

ARG BUILD_TAG

LABEL \
    com.redhat.component="rhacs-scanner-v4-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="This image supports image scanning v4 for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="This image supports image scanning v4 for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="scanner-v4" \
    io.openshift.tags="rhacs,scanner-v4,stackrox" \
    maintainer="Red Hat, Inc." \
    name="advanced-cluster-security/rhacs-scanner-v4-rhel9" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="The image scanner v4 for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${BUILD_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

COPY --from=builder \
    /src/scanner/image/scanner/scripts/entrypoint.sh \
    /src/scanner/image/scanner/scripts/import-additional-cas \
    /src/scanner/image/scanner/scripts/restore-all-dir-contents \
    /src/scanner/image/scanner/scripts/save-dir-contents \
    /src/scanner/image/scanner/bin/scanner \
    /usr/local/bin/

# The mapping files are not optional.
# The helm chart hard codes in the indexer config the path to the mapping
# files.  If the file does not exist, the indexer raises an error during bootstrap.
# (Note that the file is downloaded from Central after initial seeding.)

COPY .konflux/scanner-data/repository-to-cpe.json .konflux/scanner-data/container-name-repos-map.json /run/mappings/

COPY --from=package_installer /out/ /

RUN \
    chown -R 65534:65534 /tmp && \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 65534:65534 /etc/pki/ca-trust && save-dir-contents /etc/pki/ca-trust/source

COPY LICENSE /licenses/LICENSE

# This is equivalent to nobody:nobody.
USER 65534:65534

ENTRYPOINT ["entrypoint.sh"]
