#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

temp_dir="$(mktemp -d)"

"${SCRIPT_DIR}/../../../tests/scripts/setup-certs.sh" "$temp_dir" "custom-tls-cert.central.stackrox.local"
mv "${temp_dir}/tls.crt" "${SCRIPT_DIR}/cert-chain.pem"
rm -rf "$temp_dir"

date -u '+%Y-%m-%dT%TZ' >"${SCRIPT_DIR}/verification-time"
