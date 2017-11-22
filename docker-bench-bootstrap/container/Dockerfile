FROM scratch

ENV PATH=/

COPY ./bin/docker-bench-bootstrap /docker-bench-bootstrap

HEALTHCHECK CMD exit 0

ENTRYPOINT ["/docker-bench-bootstrap"]
