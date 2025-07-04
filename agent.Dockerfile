FROM registry.redhat.io/ubi9-minimal:9.6

COPY bin/agent /usr/local/bin
ENTRYPOINT /usr/local/bin/agent
