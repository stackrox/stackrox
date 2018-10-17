#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

oc secrets add serviceaccount/clairify secrets/stackrox --for=pull

oc create -f "$DIR/clairify.yaml"
