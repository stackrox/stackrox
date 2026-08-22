FROM golang:1.25 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /clair-adapter ./clair-adapter/cmd/clair-adapter/

FROM registry.access.redhat.com/ubi9/ubi-micro:latest
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /clair-adapter /usr/local/bin/clair-adapter
USER 65534:65534
ENTRYPOINT ["clair-adapter"]
