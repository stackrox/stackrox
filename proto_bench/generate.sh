#!/bin/bash

PB_REL="https://github.com/protocolbuffers/protobuf/releases"
curl -LO $PB_REL/download/v3.15.8/protoc-3.15.8-linux-x86_64.zip
unzip protoc-3.15.8-linux-x86_64.zip -d $PWD

mkdir -p bin
export GOBIN=$PWD/bin

go install github.com/gogo/protobuf/protoc-gen-gofast@latest

mkdir -p vtproto
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@latest
./bin/protoc \
    --go_out=vtproto --plugin protoc-gen-go="${GOBIN}/protoc-gen-go" \
    --go-vtproto_out=vtproto --plugin protoc-gen-go-vtproto="${GOBIN}/protoc-gen-go-vtproto" \
    --go-vtproto_opt=features=marshal+unmarshal+size+equal+clone+pool \
    proto/cluster.proto;

mkdir -p csproto
go install github.com/CrowdStrike/csproto/cmd/protoc-gen-fastmarshal@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
./bin/protoc \
    --go_out=csproto \
    --fastmarshal_out=apiversion=v2,paths=source_relative:csproto \
    proto/cluster.proto;

mkdir -p gogo
go install github.com/gogo/protobuf/protoc-gen-gofast@latest
${GOBIN}/protoc --gofast_out=plugins=grpc:gogo proto/cluster.proto