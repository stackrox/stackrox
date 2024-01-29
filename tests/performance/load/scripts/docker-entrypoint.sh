#!/usr/bin/env bash

# Run tests
echo "Starting k6 test '${TEST_FILE}' ..."
k6 run "${TEST_FILE}" --vus "${VUS}" --iterations "${ITERATIONS}" --duration "${DURATION}" --out csv="${OUTPUT_CSV}"

# Output result of test execution
cat "${OUTPUT_CSV}"

# Freeze return from script to keep output of k6 visible via logs and avoid restarts of container
echo "Test finished!"
tail -f /dev/null
