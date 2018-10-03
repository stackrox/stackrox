#!/bin/sh

read PASSWORD

# Docker uses URL-safe Base64 for registry-auth secrets.
# See https://github.com/golang/go/blob/8919fe9e/src/encoding/base64/base64.go#L35-L36
# for the encoding used in Go.
# The value must not have newlines.
echo "{\"username\": \"$1\", \"password\": \"$PASSWORD\"}" | base64 | tr '+/' '-_' | tr -d '\n'
