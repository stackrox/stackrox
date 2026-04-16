#!/usr/bin/env bash

# This script downloads additional components for the scanner-v4 image.
# When DEBUG_BUILD=yes, it installs the Delve debugger.

set -euo pipefail

arch=$(uname -m)
goarch="$arch"
if [[ "$arch" == "x86_64" ]]; then
    goarch="amd64"
elif [[ "$arch" == "aarch64" ]]; then
    goarch="arm64"
elif [[ "$arch" == "arm64" ]]; then
    arch="aarch64"
fi

output_dir="/output"
mkdir -p "${output_dir}/go/bin"
if [[ "$DEBUG_BUILD" == "yes" ]]; then
    if [[ "$goarch" == "amd64" || "$goarch" == "arm64" ]]; then
        dnf install -y golang
        if [[ "$OSTYPE" != "linux-gnu"* ]]; then
            GOBIN='' GOOS=linux GOARCH="${goarch}" GOPATH="${output_dir}/go" go install github.com/go-delve/delve/cmd/dlv@latest
            mv "${output_dir}/go/bin/linux_${goarch}/dlv" "${output_dir}/go/bin/dlv"
            rm -r "${output_dir}/go/bin/linux_${goarch}"
        else
            GOBIN="${output_dir}/go/bin" go install github.com/go-delve/delve/cmd/dlv@latest
        fi
    else
        echo "WARNING: Architecture ${goarch} is not supported by delve. Debugging won't be available"
    fi
fi
