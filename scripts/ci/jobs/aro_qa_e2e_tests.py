#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in an ARO cluster provided via automation-flavors/aro.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"

os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
