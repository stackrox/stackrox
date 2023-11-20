#!/bin/bash

base_url="https://services.nvd.nist.gov/rest/json/cves/2.0"
results_per_page=2000
dir_name="nvd-data"

# Function to fetch data for a given period
fetch_data_for_period () {
  local period_start_date=$1
  local period_end_date=$2
  local current_start_index=0

  while : ; do
    local file_name="${period_start_date}_${current_start_index}.json"

    echo "Fetching data from ${period_start_date} to ${period_end_date}, starting at index ${current_start_index}..."

    retry_intervals=(3 6 12) # Array of retry intervals in seconds
    success=false

    for i in "${retry_intervals[@]}"; do
        if curl -s --fail --show-error -H "apiKey: $API_KEY" "$base_url?pubStartDate=$period_start_date&pubEndDate=$period_end_date&startIndex=$current_start_index" -o "$dir_name/$file_name"; then
            success=true
            break # Exit
        else
            sleep "$i" # Wait for next retry
        fi
    done

    if [ "$success" = false ]; then
        echo "Failed after retries"
        exit 1
    fi

    if [ ! -s "$dir_name/$file_name" ] || ! jq empty "$dir_name/$file_name"; then
      echo "Downloaded file is empty or invalid for the period: $period_start_date to $period_end_date, starting index $current_start_index"
      exit 1
    fi

    local total_results
    total_results=$(jq '.totalResults' "$dir_name/$file_name") || { echo "jq command failed"; exit 1; }

    if ((current_start_index >= total_results)); then
      break
    fi

    current_start_index=$((current_start_index + results_per_page))
    sleep 3
  done
}

mkdir "$dir_name"

for (( year=2008; year<=$(date +%Y); year++ )); do
    # NVD limits data fetch to 120 days per request. We fetch quarterly to comply with this limit.
    periods=(
        "${year}-01-01T00:00:00.000 ${year}-03-31T23:59:59.999"
        "${year}-04-01T00:00:00.000 ${year}-06-30T23:59:59.999"
        "${year}-07-01T00:00:00.000 ${year}-09-30T23:59:59.999"
        "${year}-10-01T00:00:00.000 ${year}-12-31T23:59:59.999"
    )

    for period in "${periods[@]}"; do
        START_DATE=$(echo "$period" | cut -d ' ' -f1)
        END_DATE=$(echo "$period" | cut -d ' ' -f2)

        echo "Fetching data from $START_DATE to $END_DATE"
        fetch_data_for_period "$START_DATE" "$END_DATE"
        sleep 3
    done

    file_count=$(find "$dir_name" -maxdepth 1 -type f -name "*${year}*.json" | wc -l)
    echo "There are $file_count files in the $dir_name directory for year $year."

    if [ "$file_count" -gt 0 ]; then
        if jq -cs '{ vulnerabilities: map(.vulnerabilities) | add }' "$dir_name"/*"${year}"*.json > "$dir_name/combined_${year}.json"; then
            echo "The consolidated file size for $year is $(stat -c %s "$dir_name/combined_${year}.json") bytes."

            # Delete all files except 'combined_*.json' files
            find "$dir_name" -maxdepth 1 -type f ! -name 'combined_*.json' -exec rm {} \;
        else
            echo "Failed to combine JSON files for $year."
            exit 1
        fi
    else
        echo "No files to combine for year $year."
    fi
    sleep 3
done

if tar -czf "${dir_name}.tar.gz" -C "$dir_name" .; then
    echo "The size of the compressed file ${dir_name}.tar.gz is $(stat -c %s "${dir_name}.tar.gz") bytes."
else
    echo "Failed to compress the directory."
    exit 1
fi

mkdir "nvd-bundle" && mv "${dir_name}.tar.gz" "nvd-bundle/${dir_name}.tar.gz"
if ! gsutil cp -r "nvd-bundle" "gs://scanner-v4-test/"; then
    echo "gsutil upload failed"
    exit 1
fi

