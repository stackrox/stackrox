FROM scratch
MAINTAINER StackRox <support@stackrox.com>

COPY ./bin/mock-grpc-server /
EXPOSE 9090
USER 1000:1000
ENTRYPOINT ["/mock-grpc-server"]
