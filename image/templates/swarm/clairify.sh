#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

docker stack deploy -c "${DIR}/clairify.yaml" prevent --with-registry-auth
