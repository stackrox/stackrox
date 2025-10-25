FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_golang_1.24@sha256:beed4519c775d6123c11351048be29e6f93ab0adaea2c7d55977b445966f5b27 AS builder

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


FROM registry.access.redhat.com/ubi8-minimal:latest@sha256:43dde01be4e94afd22d8d95ee8abcc9f610b4e50aff5bcc141b558c74d4c68b5

ARG BUILD_TAG

LABEL \
    com.redhat.component="rhacs-scanner-v4-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="This image supports image scanning v4 for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="This image supports image scanning v4 for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="scanner-v4" \
    io.openshift.tags="rhacs,scanner-v4,stackrox" \
    maintainer="Red Hat, Inc." \
    name="advanced-cluster-security/rhacs-scanner-v4-rhel8" \
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

RUN microdnf clean all && \
    # (Optional) Remove line below to keep package management utilities
    rpm -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum && \
    chown -R 65534:65534 /tmp && \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 65534:65534 /etc/pki/ca-trust /etc/ssl && save-dir-contents /etc/pki/ca-trust /etc/ssl

COPY LICENSE /licenses/LICENSE

# This is equivalent to nobody:nobody.
USER 65534:65534

ENTRYPOINT ["entrypoint.sh"]
