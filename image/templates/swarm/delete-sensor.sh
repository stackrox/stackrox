#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

docker service rm prevent_sensor
docker secret rm prevent_registry_auth prevent_sensor_certificate prevent_sensor_private_key
