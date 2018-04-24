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
}

main "$@"
