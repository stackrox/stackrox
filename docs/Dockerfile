FROM quay.io/stackrox-io/apollo-ci:stackrox-test-0.3.45-3-gaa456c65e1 AS builder

WORKDIR /workspace
COPY content content
COPY tools tools
COPY Makefile ./

RUN make

FROM registry.access.redhat.com/ubi8-minimal:latest

# Update RPMs to hopefully aid with vulns that aren't yet patched in the base image. Also remove package management
# utils because our own policies don't recommend having them in the image. For that we need to be root.
RUN microdnf upgrade && \
    microdnf install \
        python39 && \
    microdnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

WORKDIR /public
COPY --from=builder /workspace/build/ ./
EXPOSE 8080
CMD python -m http.server 8080
