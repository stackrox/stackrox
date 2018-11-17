#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

oc delete -f "$DIR/sensor.yaml"
oc delete -n {{.Namespace}} secret/sensor-tls