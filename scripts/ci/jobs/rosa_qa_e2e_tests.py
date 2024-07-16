#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a ROSA cluster provided via automation-flavors/rosa.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["LOAD_BALANCER"] = "route"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"
if "-hcp-" in os.environ["JOB_NAME"]:
    os.environ["MANAGED_CP"] = "true"
    # ROX-22448
    os.environ["DISABLE_AUDIT_LOG_ALERTS_TEST"] = "true"
    # ROX-22502 - ROSA HCP is missing LoadBalancer support
    os.environ["SUPPORTS_LOAD_BALANCER_SVC"] = "false"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
