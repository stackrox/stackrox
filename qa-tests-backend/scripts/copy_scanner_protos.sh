#! /bin/bash

JAVA_PATH=src/main/proto/

# Migrate v1 API protos from the Scanner repo
SCANNER_DIR=$(go list -f '{{.Dir}}' -m github.com/stackrox/scanner)
# files from gomod cache have no write permission causing problems for gradle
cp -r ${SCANNER_DIR}/proto/scanner ${JAVA_PATH}
chmod -R u+w ${JAVA_PATH}/scanner
