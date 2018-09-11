#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

WD=$(pwd)
cd "$DIR"

docker stack deploy -c ./central.yaml prevent --with-registry-auth

cd "$WD"
