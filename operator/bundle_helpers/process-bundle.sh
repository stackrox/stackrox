#!/bin/bash
set -euo pipefail

# Script to handle common bundle processing logic
# Usage: process-bundle.sh --use-version=VERSION --first-version=VERSION --related-images-mode=MODE --operator-image=IMAGE [--output-dir=DIR] [--unreleased=VERSION]

# Default values
OUTPUT_DIR="build/bundle"
ARGS=()

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --output-dir=*)
            OUTPUT_DIR="${1#*=}"
            shift
            ;;
        --use-version=*|--first-version=*|--related-images-mode=*|--operator-image=*|--unreleased=*)
            ARGS+=("$1")
            shift
            ;;
        *)
            echo "Unknown option $1" >&2
            exit 1
            ;;
    esac
done

# Validate required parameters
required_params=("--use-version" "--first-version" "--related-images-mode" "--operator-image")
for param in "${required_params[@]}"; do
    found=false
    if [[ ${#ARGS[@]} -gt 0 ]]; then
        for arg in "${ARGS[@]}"; do
            if [[ "$arg" == "$param="* ]]; then
                found=true
                break
            fi
        done
    fi
    if [[ "$found" == false ]]; then
        echo "Error: $param parameter is required" >&2
        exit 1
    fi
done

# Ensure output directory exists and copy CRD
mkdir -p "$OUTPUT_DIR/manifests"
cp -v "../config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml" "$OUTPUT_DIR/manifests/"

# Call patch-csv.py with collected arguments
"$(dirname "$0")/patch-csv.py" "${ARGS[@]}" \
    < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
    > "$OUTPUT_DIR/manifests/rhacs-operator.clusterserviceversion.yaml"

echo "Bundle processing completed successfully"