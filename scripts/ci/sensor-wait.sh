#!/bin/sh
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
    until [ "$(kubectl get pod -n stackrox --selector 'app=sensor' | grep Running | wc -l)" -eq 1 ]; do
        echo -n .
        sleep 1
    done
    echo "Sensor is running"

    if ! kubectl -n stackrox get ds/collector ; then
        return
    fi

    echo "Waiting for collectors to start"
    until [ "$(kubectl -n stackrox get po --selector 'service=collector' | tail -n +2 | awk '{print $3}' | grep Running | wc -l)" -eq "$(kubectl get nodes | tail -n +2 | wc -l)" ]; do
        echo -n .
        sleep 1
    done
    echo "Collectors are running"
}

main "$@"
