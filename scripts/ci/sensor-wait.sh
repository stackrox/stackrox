#!/usr/bin/env bash
set -eu

# Wait for sensor to be up
#
#
#
# Usage:
#   sensor-wait.sh
#
# Example:
# $ ./scripts/ci/sensor-wait.sh
#

main() {
    echo "Waiting for sensor to start"
    start_time="$(date '+%s')"
    until [ "$(kubectl get pod -n stackrox --selector 'app=sensor' | grep Running | wc -l)" -eq 1 ]; do
    	if (( $(date '+%s') - start_time > 300 )); then
    		echo >&2 "Timed out after 5m"
    		exit 1
    	fi
        echo -n .
        sleep 1
    done
    echo "Sensor is running"

    if ! kubectl -n stackrox get ds/collector ; then
        return
    fi

    echo "Waiting for collectors to start"
    start_time="$(date '+%s')"
    until [ "$(kubectl -n stackrox get ds/collector | tail -n +2 | awk '{print $2}')" -eq "$(kubectl -n stackrox get ds/collector | tail -n +2 | awk '{print $4}')" ]; do
    	if (( $(date '+%s') - start_time > 300 )); then
    		echo >&2 "Timed out after 5m"
    		exit 1
    	fi
        echo -n .
        sleep 1
    done
    echo "Collectors are running"
}

main "$@"
