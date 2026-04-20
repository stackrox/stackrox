#!/bin/bash
# Helper script to test a single option across all applicable modes

set -e

OPTION_NAME="$1"
OPTION_VALUE="$2"
MODES="$3"  # Comma-separated: openshift-pvc,k8s-pvc,openshift-hostpath,k8s-hostpath

if [ -z "$OPTION_NAME" ] || [ -z "$OPTION_VALUE" ] || [ -z "$MODES" ]; then
    echo "Usage: $0 <option-name> <option-value> <modes>"
    echo "Example: $0 db-name custom-db-name openshift-pvc,k8s-pvc"
    exit 1
fi

BASEDIR="/home/mowsiany/go/src/github.com/stackrox/stackrox/MIGRATION"
cd "$BASEDIR"

IFS=',' read -ra MODE_ARRAY <<< "$MODES"

for MODE in "${MODE_ARRAY[@]}"; do
    # Parse mode
    if [[ "$MODE" == *"openshift"* ]]; then
        PLATFORM="openshift"
    else
        PLATFORM="k8s"
    fi

    if [[ "$MODE" == *"pvc"* ]]; then
        STORAGE="pvc"
    else
        STORAGE="hostpath"
    fi

    OUTPUT_DIR="test-outputs/${OPTION_NAME}-${MODE}"

    echo "Testing --${OPTION_NAME}=${OPTION_VALUE} on ${MODE}..."

    # Generate manifests
    roxctl central generate "$PLATFORM" "$STORAGE" \
        "--${OPTION_NAME}=${OPTION_VALUE}" \
        --output-dir "$OUTPUT_DIR" > /dev/null 2>&1

    # Create diff
    DIFF_FILE="diffs/${OPTION_NAME}-${MODE}.diff"
    diff -ru "baselines/${MODE}" "$OUTPUT_DIR" > "$DIFF_FILE" 2>&1 || true

    # Show changed files (excluding random elements)
    echo "Changed files:"
    diff -qr "baselines/${MODE}" "$OUTPUT_DIR" 2>&1 | \
        grep -v 'password\|tls-secret\|htpasswd' | \
        sed 's|baselines/[^/]*/||g; s|test-outputs/[^/]*/||g' || echo "  (only random elements)"

    echo "Diff saved to: $DIFF_FILE"
    echo ""
done
