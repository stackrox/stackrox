#!/usr/bin/env bash
set -e

function is_openshift {
    kubectl get scc > /dev/null 2>&1
}
