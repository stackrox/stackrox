#!/bin/bash

# Function to download a file with curl and handle errors
download_file() {
    local url=$1
    local output_path=$2

    curl --fail --silent --show-error --max-time 60 --retry 3 -o "$output_path" "$url" || (echo "Error fetching file from $url" && exit 1)
}

# First argument: Year
YEAR=$1

# Paths
META_PATH="nvddata/nvdcve-1.1-${YEAR}.meta"
JSON_GZ_PATH="nvddata/nvdcve-1.1-${YEAR}.json.gz"
JSON_PATH="nvddata/nvdcve-1.1-${YEAR}.json"
NVD_URL="https://nvd.nist.gov/feeds/json/cve/1.1/nvdcve-1.1"

# Create the nvddata directory
mkdir -p nvddata

# Fetch the meta file for the given year
download_file "$NVD_URL"-"$YEAR".meta "$META_PATH"

CHECKSUM_META=$(grep 'sha256' "$META_PATH" | cut -d':' -f2 | tr -d '[:space:]')

# Download the .json.gz file
download_file "$NVD_URL"-"${YEAR}".json.gz "$JSON_GZ_PATH"

gunzip -c "$JSON_GZ_PATH" > "$JSON_PATH"

CHECKSUM_DOWNLOADED=$(sha256sum "$JSON_PATH" | cut -d' ' -f1 | tr 'a-f' 'A-F')

rm "$JSON_PATH"

# Verify integrity
if [[ "$CHECKSUM_META" != "$CHECKSUM_DOWNLOADED" ]]; then
    echo "Checksum verification failed for year $YEAR"
    exit 1
fi

gsutil cp -r "nvddata" "gs://scanner-v4-test/"

# Check the exit status of the gsutil command
if [ $? -ne 0 ]; then
    echo "gsutil upload failed"
    exit 1
fi
