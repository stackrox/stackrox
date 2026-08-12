#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in an OCP cluster.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster
from common import enable_sfa_for_ocp

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["KUBERNETES_PROVIDER"] = "ocp"
# Workload identities are only set up for `openshift-4` infra clusters.
if 'openshift-4' in os.environ.get('CLUSTER_FLAVOR_VARIANT', ''):
    os.environ["SETUP_WORKLOAD_IDENTITIES"] = "true"

enable_sfa_for_ocp()

os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# EXPERIMENT: exclude rhel-vex (51.7 MB, largest bundle) to identify which
# tests depend on it. If tests pass without it, this bundle is unnecessary
# for QA e2e and can be permanently excluded — saving ~5 min of import time.
# See bundle sizes in gs://definitions.stackrox.io/v4/vulnerability-bundles/v3/
#   rhel-vex:          51.7 MB (EXCLUDED)
#   nvd:               36.8 MB
#   osv:               27.0 MB
#   ubuntu:            18.5 MB
#   debian:            13.1 MB
#   epss:               4.3 MB
#   stackrox-rhel-csaf: 0.8 MB
#   manual:             0.0 MB
#   alpine:             1.1 MB (EXCLUDED)
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = "alpine,debian,epss,manual,nvd,osv,stackrox-rhel-csaf,ubuntu"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
