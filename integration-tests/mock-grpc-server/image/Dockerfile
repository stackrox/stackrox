FROM scratch
LABEL name="mock-grpc-server" \
      maintainer="https://stackrox.io/"
ARG TARGETARCH
ARG TARGETOS

COPY ./bin/${TARGETOS}_${TARGETARCH}/mock-grpc-server /
EXPOSE 9090
USER 1000:1000
ENTRYPOINT ["/mock-grpc-server"]
