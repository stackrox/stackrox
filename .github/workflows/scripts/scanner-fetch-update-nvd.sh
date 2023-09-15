#!/usr/bin/env bash
set -eu
# set -x  # Uncomment this if you want verbose debugging output

# This script fetches NVD CVE data for a given year, verifies its integrity, and uploads it to a GCS bucket.

# ------------------------ Instructions ------------------------
# 1. Ensure you have `curl`, `gunzip`, `sha256sum` and `gsutil` installed.
# 2. Run this script with a year as an argument: `./<script_name>.sh 2023`
# 3. Check the script logs for any errors. The script will stop execution if there's an error.

# Function to download a file with curl and handle errors
download_file() {
    local url=$1
    local output_path=$2

    if ! curl --fail --silent --show-error --max-time 60 --retry 3 -o "$output_path" "$url"; then
        echo "Error fetching file from $url"
        exit 1
    fi
}


# First argument: Year
YEAR=${1:-}
if [ -z "$YEAR" ]; then
    echo >&2 "error: missing YEAR argument."
    exit 1
fi

# Paths
META_PATH="nvddata/nvdcve-1.1-${YEAR}.meta"
JSON_GZ_PATH="nvddata/nvdcve-1.1-${YEAR}.json.gz"
NVD_URL="https://nvd.nist.gov/feeds/json/cve/1.1/nvdcve-1.1"

# Create the nvddata directory
mkdir -p nvddata

# Fetch the meta file for the given year
download_file "$NVD_URL"-"$YEAR".meta "$META_PATH"

CHECKSUM_META=$(grep 'sha256' "$META_PATH" | cut -d':' -f2 | tr -d '[:space:]')

# Download the .json.gz file
download_file "$NVD_URL"-"${YEAR}".json.gz "$JSON_GZ_PATH"

CHECKSUM_DOWNLOADED=$(gzip -dc "$JSON_GZ_PATH" | sha256sum | cut -d' ' -f1 | tr 'a-f' 'A-F')

# Verify integrity
if [[ "$CHECKSUM_META" != "$CHECKSUM_DOWNLOADED" ]]; then
    echo "Checksum verification failed for year $YEAR"
    exit 1
fi

if ! gsutil cp -r "nvddata" "gs://scanner-v4-test/"; then
    echo "gsutil upload failed"
    exit 1
fi
