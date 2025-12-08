#!/usr/bin/env bash

# Convenient wrapper for Bazel builds in StackRox
# Usage: ./bzl <binary_name> [platform]
#
# Examples:
#   ./bzl roxctl
#   ./bzl roxctl linux_arm64
#   ./bzl migrator
#   ./bzl admission-control
#
# This is a convenience wrapper to avoid typing long bazelisk commands

set -euo pipefail

BINARY="${1:-}"
PLATFORM="${2:-linux_amd64}"

if [ -z "$BINARY" ]; then
    cat <<EOF
Usage: ./bzl <binary> [platform]

Supported binaries:
  roxctl              - CLI tool (97MB)
  migrator            - Database migrations (79MB)
  admission-control   - Admission controller (101MB)
  upgrader            - Sensor upgrader (90MB)
  init-tls-certs      - TLS certificate init (1.7MB)
  config-controller   - Config controller (76MB)
  compliance          - Compliance binary (406KB)

Supported platforms:
  linux_amd64   (default)
  linux_arm64
  darwin_amd64
  darwin_arm64
  windows_amd64

Not yet supported (use Make):
  central             - Use: make central-build-nodeps
  kubernetes          - Use: make sensor-build

Examples:
  ./bzl roxctl
  ./bzl roxctl darwin_arm64
  ./bzl migrator
  ./bzl admission-control

For full command:
  bazelisk build //roxctl:roxctl --config=linux_amd64

For more info:
  cat BAZEL_QUICKSTART.md
EOF
    exit 1
fi

# Map friendly names to Bazel targets
case "$BINARY" in
    roxctl)
        TARGET="//roxctl:roxctl"
        ;;
    migrator)
        TARGET="//migrator:migrator"
        ;;
    admission-control|admission)
        TARGET="//sensor/admission-control:admission-control"
        ;;
    upgrader)
        TARGET="//sensor/upgrader:upgrader"
        ;;
    init-tls-certs|init-tls)
        TARGET="//sensor/init-tls-certs:init-tls-certs"
        ;;
    config-controller|config)
        TARGET="//config-controller:config-controller"
        ;;
    compliance)
        TARGET="//compliance:compliance"
        ;;
    central)
        echo "❌ ERROR: Central is not yet supported with Bazel (scanner dependency issue)"
        echo "Use: make central-build-nodeps"
        exit 1
        ;;
    kubernetes|sensor)
        echo "❌ ERROR: Sensor/Kubernetes is not yet supported with Bazel (scanner dependency issue)"
        echo "Use: make sensor-build"
        exit 1
        ;;
    *)
        echo "❌ ERROR: Unknown binary: $BINARY"
        echo "Run './bzl' for list of supported binaries"
        exit 1
        ;;
esac

echo "Building $BINARY for $PLATFORM..."
echo "Command: bazelisk build $TARGET --config=$PLATFORM"
echo ""

# Run the build
bazelisk build "$TARGET" --config="$PLATFORM"

# Show the output location using deterministic bazel-bin path
# This avoids running cquery which invalidates the analysis cache
# Parse target format: //package/path:target_name
TARGET_NO_SLASHES="${TARGET#//}"  # Remove leading //
PACKAGE_PATH="${TARGET_NO_SLASHES%:*}"  # Everything before :
BINARY_NAME="${TARGET_NO_SLASHES##*:}"  # Everything after :

# Construct the path: bazel-bin/<package_path>/<target_name>_/<binary_name>
BINARY_PATH="bazel-bin/${PACKAGE_PATH}/${BINARY_NAME}_/${BINARY_NAME}"

if [ -f "$BINARY_PATH" ]; then
    echo ""
    echo "✅ Build successful!"
    echo "Binary location: $BINARY_PATH"
    ls -lh "$BINARY_PATH"
fi

