FROM quay.io/stackrox-io/apollo-ci:stackrox-test-0.3.49 AS builder

WORKDIR /workspace
COPY content content
COPY tools tools
COPY Makefile ./
RUN wget -O busybox https://busybox.net/downloads/binaries/1.31.0-defconfig-multiarch-musl/busybox-`arch` && chmod +x busybox

RUN make

FROM registry.access.redhat.com/ubi8-micro:latest

WORKDIR /public
COPY --from=builder /workspace/build/ ./
COPY --from=builder /workspace/busybox /bin/busybox
EXPOSE 8080
CMD busybox httpd -f -p 8080
