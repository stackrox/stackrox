#!/bin/bash


base_url="https://services.nist.gov/rest/json/cves/2.0"
severity="CRITICAL"
results_per_page=2000
start_index=0
total_results=1  # Initial dummy value to enter the loop

# Function to fetch data and return the next startIndex
fetch_data () {
  # Fetch data with curl and check the exit status directly
  if ! response=$(curl -s "$base_url?cvssV3Severity=$severity&resultsPerPage=$results_per_page&startIndex=$start_index"); then
    status=$?
    echo "Curl failed with status $status"
    exit $status
  fi


  # Extract the total number of results from the response
  if [ $start_index -eq 0 ]; then  # Only do this the first time.
    total_results=$(echo "$response" | jq '.totalResults')
  fi

  next_index=$(echo "$response" | jq '.startIndex + .resultsPerPage')

  echo "$response" > "critical-$((start_index/results_per_page + 1)).json"

  # Output the next start index
  echo "$next_index"
}

while [ "$start_index" -lt "$total_results" ]; do
  echo "Fetching records starting from index $start_index..."
  start_index=$(fetch_data)  # Fetch the data and get the next start index

  if [ $? -ne 0 ]; then
    exit $?
  fi

  # Sleep for 6 seconds before the next request
  echo "Waiting for 6 seconds..."
  sleep 6
done

echo "Done. All critical CVEs are stored in separate files."
