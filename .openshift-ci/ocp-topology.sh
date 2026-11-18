#!/usr/bin/env bash

# Set topology before automation-flavors sources shared_env.
configure_ocp_topology() {
    if [[ ! "${JOB_NAME:-}" =~ -ocp- ]]; then
        return 0
    fi

    if [[ "${JOB_NAME:-}" =~ -ocp-4-22-(operator-e2e-tests|nongroovy-e2e-tests|scanner-v4-install-tests|qa-e2e-tests|ui-e2e-tests)$ ]]; then
        # Start with the documented GCP SNO path; size from PR CI results.
        set_ci_shared_export MASTER_NODE_COUNT 1
        set_ci_shared_export MASTER_NODE_TYPE e2-standard-16
        set_ci_shared_export WORKER_NODE_COUNT 0
    else
        set_ci_shared_export WORKER_NODE_COUNT 2
    fi

    if [[ "${JOB_NAME:-}" =~ vm-scanning ]]; then
        set_ci_shared_export WORKER_NODE_TYPE n2-standard-8
    else
        set_ci_shared_export WORKER_NODE_TYPE e2-standard-8
    fi
    info "OCP masters=${MASTER_NODE_COUNT:-provisioner-default} master-type=${MASTER_NODE_TYPE:-provisioner-default} workers=$WORKER_NODE_COUNT worker-type=$WORKER_NODE_TYPE"
}
