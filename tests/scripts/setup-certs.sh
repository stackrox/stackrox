#!/usr/bin/env bash

set -euo pipefail

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

# shellcheck source=../../scripts/lib.sh
source "$TEST_ROOT/scripts/lib.sh"

setup_certs() {
    target_dir="$1"
    cn="$2"

    [[ -n "$cn" ]] || {
        echo >&2 "No CN specified!"
        exit 1
    }

    ca_name="${3:-Test CA}"

    [[ -d "$target_dir" ]] || mkdir "$target_dir"
    chmod 0700 "$target_dir"

    pushd "$target_dir"

    # Extensions for intermediate CA
    intermediate_ca_exts='
basicConstraints = critical, CA:true, pathlen:0
keyUsage = critical, digitalSignature, cRLSign, keyCertSign
'

    root_ca_exts="
  [req]
  distinguished_name=dn
  x509_extensions=ext
  [ dn ]
  [ ext ]
  basicConstraints=CA:TRUE,pathlen:1
  "
    # Root CA
    openssl req -nodes -config <(echo "$root_ca_exts") -new -x509 -keyout ca.key -out ca.crt -subj "/CN=Root ${ca_name}"

    # Intermediate CA
    openssl genrsa -out intermediate.key 4096
    openssl req -new -key intermediate.key -subj "/CN=Intermediate ${ca_name}" |
        openssl x509 -sha256 -extfile <(echo "$intermediate_ca_exts") -req -CA ca.crt -CAkey ca.key -CAcreateserial -out intermediate.crt

    leaf_ca_exts="subjectAltName=DNS:${cn}"

    # Leaf cert
    openssl genrsa -out leaf.key 4096
    openssl req -new -key leaf.key -subj "/CN=${cn}" |
        openssl x509 -sha256 -extfile <(echo "$leaf_ca_exts") -req -CA intermediate.crt -CAkey intermediate.key -CAcreateserial -out leaf.crt

    cat leaf.crt intermediate.crt ca.crt >tls.crt
    cp leaf.key tls.key

    openssl pkcs12 -export -inkey tls.key -in tls.crt -out keystore.p12 -passout pass:

    popd
}

# shellcheck disable=SC2120
setup_default_TLS_certs() {
    info "Setting up default certs for tests"

    local cert_dir
    cert_dir="${1:-$(mktemp -d)}"
    setup_certs "$cert_dir" custom-tls-cert.central.stackrox.local "Server CA"

    export_default_TLS_certs "${cert_dir}"
}

export_default_TLS_certs() {
    local cert_dir="$1"
    
    export ROX_DEFAULT_TLS_CERT_FILE="${cert_dir}/tls.crt"
    export ROX_DEFAULT_TLS_KEY_FILE="${cert_dir}/tls.key"
    export DEFAULT_CA_FILE="${cert_dir}/ca.crt"
    ROX_TEST_CA_PEM="$(cat "${cert_dir}/ca.crt")"
    export ROX_TEST_CA_PEM="$ROX_TEST_CA_PEM"
    export ROX_TEST_CENTRAL_CN="custom-tls-cert.central.stackrox.local"
    export TRUSTSTORE_PATH="${cert_dir}/keystore.p12"

    echo "Contents of ${cert_dir}:"
    ls -al "${cert_dir}"
}

# shellcheck disable=SC2120
setup_client_TLS_certs() {
    info "Setting up client certs for tests"

    local cert_dir
    cert_dir="${1:-$(mktemp -d)}"
    setup_certs "$cert_dir" "Client Certificate User" "Client CA"

    export_client_TLS_certs "${cert_dir}"
}

export_client_TLS_certs() {
    local cert_dir="$1"
    
    export KEYSTORE_PATH="$cert_dir/keystore.p12"
    export CLIENT_CA_PATH="$cert_dir/ca.crt"
    export CLIENT_CERT_PATH="$cert_dir/tls.crt"
    export CLIENT_KEY_PATH="$cert_dir/tls.key"

    echo "Contents of ${cert_dir}:"
    ls -al "${cert_dir}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    setup_certs "$@"
fi
