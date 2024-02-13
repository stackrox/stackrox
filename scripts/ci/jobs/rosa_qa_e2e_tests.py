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
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"
if "MANAGED_CP" in os.environ and os.environ["MANAGED_CP"] == "true":
    # ROX-22448
    os.environ["DISABLE_AUDIT_LOG_ALERTS_TEST"] = "true"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
