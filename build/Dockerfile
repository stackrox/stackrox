FROM golang:1.13.9-alpine3.11 as builder

ENV ROCKSDB_VERSION="v6.7.3"

RUN apk add --update --no-cache build-base linux-headers git cmake bash
RUN apk add --update --no-cache zlib zlib-dev bzip2 bzip2-dev snappy snappy-dev lz4 lz4-dev zstd zstd-dev perl

# installing latest gflags
RUN cd /tmp && \
    git clone https://github.com/gflags/gflags.git && \
    cd gflags && \
    mkdir build && \
    cd build && \
    cmake -DBUILD_SHARED_LIBS=1 -DGFLAGS_INSTALL_SHARED_LIBS=1 .. && \
    make install && \
    make install DESTDIR=/dist

# Compile Rocksdb
RUN cd /tmp && \
    git clone -b "${ROCKSDB_VERSION}" --depth 1 https://github.com/facebook/rocksdb.git && \
    cd rocksdb && \
    make static_lib

RUN cd /tmp/rocksdb && DEBUG_LEVEL=0 make ldb

RUN wget -O- https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz | tar -xJf -
RUN upx-3.96-amd64_linux/upx -9 /tmp/rocksdb/ldb

FROM golang:1.13.9-alpine3.11

COPY --from=builder /tmp/rocksdb/librocksdb.a /lib/rocksdb/librocksdb.a
COPY --from=builder /tmp/rocksdb/include /lib/rocksdb/include
COPY --from=builder /tmp/rocksdb/ldb /usr/local/bin/ldb

ENV CGO_CFLAGS="-I/lib/rocksdb/include"
ENV CGO_LDFLAGS="-L/lib/rocksdb -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"
ENV CGO_ENABLED=1
ENV GOCACHE="/linux-gocache"

RUN apk add --update --no-cache build-base linux-headers git cmake bash
RUN apk add --update --no-cache zlib zlib-dev bzip2 bzip2-dev snappy snappy-dev lz4 lz4-dev zstd zstd-dev perl

WORKDIR /go/src/github.com/stackrox/rox
