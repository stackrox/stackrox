#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a IBM CLOUD Z Openshift cluster provided via
automation-flavors/ibmcloudz.
"""
import os
import scanner_v4_defaults
from base_qa_e2e_test import make_qa_e2e_test_runner_custom
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["LOAD_BALANCER"] = "route"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["USE_MIDSTREAM_IMAGES"] = "true"
os.environ["REMOTE_CLUSTER_ARCH"] = "s390x"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# Scanner V4
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = scanner_v4_defaults.VULN_BUNDLE_ALLOWLIST

make_qa_e2e_test_runner_custom(cluster=AutomationFlavorsCluster()).run()
