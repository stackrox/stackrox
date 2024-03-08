#! /bin/bash

JAVA_PATH=src/main/proto/

# Migrate protos from the stackrox repo.
mkdir -p ${JAVA_PATH}
cp -r ../proto/* ${JAVA_PATH}

# Migrate v1 API protos from the Scanner repo
SCANNER_DIR=$(go list -f '{{.Dir}}' -m github.com/stackrox/scanner)
SCANNER_PROTO_BASE_PATH=$SCANNER_DIR/proto/
# files from gomod cache have no write permission causing problems for gradle
cp -r --no-preserve=mode ${SCANNER_PROTO_BASE_PATH}/* ${JAVA_PATH}
