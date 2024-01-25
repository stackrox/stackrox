#! /bin/bash

JAVA_PATH=src/main/proto/

# Migrate protos from the stackrox repo.

for file in $(find ../proto/*); do
    if [[ -d $file ]]; then
        dir=${file#"../proto/"}
        mkdir -p "${JAVA_PATH}${dir}"
        echo "${JAVA_PATH}${dir}"
    fi
done

for file in $(find ../proto/* -name '*.proto'); do
    if [[ -f $file ]]; then
        java_file=${file#"../proto/"}
        sed -e 's/\[[^][]*\]//g' "$file" | sed -e 's/\[[^][]*\]//g' | sed '/gogo/d'| sed -e ':a' -e 'N' -e '$!ba' -e 's/,\(\n\s*\]\)/\1/g'> "${JAVA_PATH}${java_file}"
    fi
done

# Migrate v1 API protos from the Scanner repo

SCANNER_DIR=$(go list -f '{{.Dir}}' -m github.com/stackrox/scanner)
SCANNER_PROTO_BASE_PATH=$SCANNER_DIR/proto

mkdir -p "${JAVA_PATH}scanner/api/v1"
echo "${JAVA_PATH}scanner/api/v1"

for file in $(find "$SCANNER_PROTO_BASE_PATH" -name '*.proto'); do
    if [[ -f $file ]]; then
        # Get relative path. Should be along the lines of scanner/api/v1/*.proto
        rel_file=${file/"$SCANNER_PROTO_BASE_PATH"/""}
        rel_file="${rel_file:1}"
        sed -e 's/\[[^][]*\]//g' "$file" | sed -e 's/\[[^][]*\]//g' | sed '/gogo/d' | sed -e ':a' -e 'N' -e '$!ba' -e 's/,\(\n\s*\]\)/\1/g'> "${JAVA_PATH}${rel_file}"
    fi
done

