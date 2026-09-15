#!/usr/bin/env bash
# Script to insert the generated ImageCopies list into pkg/fixtures/image.go
# Run this after generate-test-images.sh completes
#
# Usage: ./insert-generated-images.sh [--insert]
#
# Options:
#   --insert    Automatically insert the ImageCopies list into the Go file

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
IMAGE_GO_FILE="${REPO_ROOT}/pkg/fixtures/image.go"

IMAGES_ENTRIES_FILE="${SCRIPT_DIR}/generated-images-entries.txt"
OUTPUT_SNIPPET="${SCRIPT_DIR}/generated-images-snippet.go"

INSERT=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --insert)
            INSERT=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

if [[ ! -f "${IMAGES_ENTRIES_FILE}" ]]; then
    echo "ERROR: ${IMAGES_ENTRIES_FILE} not found"
    echo "Run generate-test-images.sh first to generate the image entries"
    exit 1
fi

echo "Generating ImageCopies snippet..."

# Count entries
entry_count=$(wc -l < "${IMAGES_ENTRIES_FILE}" | tr -d ' ')
echo "Found ${entry_count} image entries"

# Generate the Go code snippet
cat > "${OUTPUT_SNIPPET}" << 'HEADER'

	// ImageCopies contains generated test images hosted on quay.io/rh_ee_chsheth/image-model-test
	// Each image has two copies tagged as <image_name>_orig and <image_name>_copy
	// These images are lightweight, built from alpine/busybox/debian-slim bases with small packages
	ImageCopies = []ImageAndID{
HEADER

# Add all entries (they already have the correct format)
while IFS= read -r line; do
    echo "		${line}" >> "${OUTPUT_SNIPPET}"
done < "${IMAGES_ENTRIES_FILE}"

echo "	}" >> "${OUTPUT_SNIPPET}"

echo ""
echo "Generated Go snippet saved to: ${OUTPUT_SNIPPET}"

if [[ "${INSERT}" == "true" ]]; then
    echo ""
    echo "Inserting ImageCopies into ${IMAGE_GO_FILE}..."
    
    # Check if ImageCopies already exists
    if grep -q "ImageCopies = \[\]ImageAndID{" "${IMAGE_GO_FILE}"; then
        echo "ERROR: ImageCopies already exists in ${IMAGE_GO_FILE}"
        echo "Please remove the existing ImageCopies list first if you want to regenerate it"
        exit 1
    fi
    
    # Find the line number of the last '}' before the final ')'
    last_brace_line=$(grep -n "^[[:space:]]*}$" "${IMAGE_GO_FILE}" | tail -1 | cut -d: -f1)
    
    # Create temp file
    temp_file=$(mktemp)
    
    # Copy everything up to and including the last }
    head -n "${last_brace_line}" "${IMAGE_GO_FILE}" > "${temp_file}"
    
    # Append the ImageCopies snippet
    cat "${OUTPUT_SNIPPET}" >> "${temp_file}"
    
    # Append the closing )
    echo ")" >> "${temp_file}"
    
    # Replace the original file
    mv "${temp_file}" "${IMAGE_GO_FILE}"
    
    echo "Successfully inserted ImageCopies into ${IMAGE_GO_FILE}"
    echo ""
    echo "Run 'gofmt -w ${IMAGE_GO_FILE}' to format the file"
else
    echo ""
    echo "To add this to ${IMAGE_GO_FILE}, run:"
    echo "  ${SCRIPT_DIR}/insert-generated-images.sh --insert"
    echo ""
    echo "Preview of the generated snippet (first 20 lines):"
    head -20 "${OUTPUT_SNIPPET}"
    echo "..."
    echo ""
    echo "Preview of the last 10 lines:"
    tail -10 "${OUTPUT_SNIPPET}"
fi
