#!/bin/bash
set -euo pipefail

# Script to handle common bundle processing logic
# Usage: process-bundle.sh --use-version=VERSION --first-version=VERSION --related-images-mode=MODE --operator-image=IMAGE [--output-dir=DIR] [--unreleased=VERSION]
# Required parameters:
#   --use-version=VERSION        Version to use for the bundle
#   --first-version=VERSION      First version for patch-csv.py
#   --related-images-mode=MODE   Related images mode (omit, konflux, etc.)
#   --operator-image=IMAGE       Operator image reference
# Optional parameters:
#   --output-dir=DIR            Output directory (default: build/bundle)
#   --unreleased=VERSION        Unreleased version for patch-csv.py

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OPERATOR_DIR="$(dirname "$SCRIPT_DIR")"

# Required parameters (no defaults)
RELATED_IMAGES_MODE=""
OPERATOR_IMAGE=""
USE_VERSION=""
FIRST_VERSION=""

# Optional parameters with defaults
OUTPUT_DIR="build/bundle"
UNRELEASED_VERSION=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --related-images-mode=*)
            RELATED_IMAGES_MODE="${1#*=}"
            shift
            ;;
        --operator-image=*)
            OPERATOR_IMAGE="${1#*=}"
            shift
            ;;
        --output-dir=*)
            OUTPUT_DIR="${1#*=}"
            shift
            ;;
        --use-version=*)
            USE_VERSION="${1#*=}"
            shift
            ;;
        --unreleased=*)
            UNRELEASED_VERSION="${1#*=}"
            shift
            ;;
        --first-version=*)
            FIRST_VERSION="${1#*=}"
            shift
            ;;
        *)
            echo "Unknown option $1"
            exit 1
            ;;
    esac
done

# Validate required parameters
if [[ -z "$USE_VERSION" ]]; then
    echo "Error: --use-version parameter is required" >&2
    exit 1
fi

if [[ -z "$RELATED_IMAGES_MODE" ]]; then
    echo "Error: --related-images-mode parameter is required" >&2
    exit 1
fi

if [[ -z "$OPERATOR_IMAGE" ]]; then
    echo "Error: --operator-image parameter is required" >&2
    exit 1
fi

if [[ -z "$FIRST_VERSION" ]]; then
    echo "Error: --first-version parameter is required" >&2
    exit 1
fi

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR/manifests"

# Copy securitypolicies CRD (the main duplication point)
cp -v "../config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml" "$OUTPUT_DIR/manifests/"

# Common patch-csv.py arguments
COMMON_ARGS=(
    "--first-version" "$FIRST_VERSION"
    "--add-supported-arch" "amd64"
    "--add-supported-arch" "arm64"
    "--add-supported-arch" "ppc64le"
    "--add-supported-arch" "s390x"
    "--related-images-mode=${RELATED_IMAGES_MODE}"
)

# Add version if provided
if [[ -n "$USE_VERSION" ]]; then
    COMMON_ARGS+=("--use-version" "$USE_VERSION")
fi

# Add operator image if provided
if [[ -n "$OPERATOR_IMAGE" ]]; then
    COMMON_ARGS+=("--operator-image" "$OPERATOR_IMAGE")
fi

# Add unreleased version if provided
if [[ -n "$UNRELEASED_VERSION" ]]; then
    COMMON_ARGS+=("--unreleased" "$UNRELEASED_VERSION")
fi

# Run patch-csv.py with common arguments
"$SCRIPT_DIR/patch-csv.py" "${COMMON_ARGS[@]}" \
    < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
    > "$OUTPUT_DIR/manifests/rhacs-operator.clusterserviceversion.yaml"

echo "Bundle processing completed successfully"