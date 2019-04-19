#!/usr/bin/env bash

set -euo pipefail

head -c 32 /dev/random >../../image/keys/data-key
head -c 16 /dev/random >../../image/keys/data-iv

cat >gen-keys.go <<EOF
// Code generate by generate-keys.sh. DO NOT EDIT.

package encdata

var (
    key = []byte{
        $(xxd -i <../../image/keys/data-key | sed '$s/$/,/')
    }
    iv = []byte{
        $(xxd -i <../../image/keys/data-iv | sed '$s/$/,/')
    }
)
EOF
