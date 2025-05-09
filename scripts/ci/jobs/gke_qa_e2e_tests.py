#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import GKECluster

# set test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

os.environ["ROX_ACTIVE_VULN_MGMT"] = "true"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# deploy via help to set node selectors
os.environ["OUTPUT_FORMAT"] = "helm"
os.environ["SENSOR_HELM_MANAGED"] = "true"
os.environ["ROX_CENTRAL_EXTRA_HELM_VALUES_FILE"] = "central-arm64-values.yaml"
os.environ["ROX_SENSOR_EXTRA_HELM_VALUES_FILE"] = "sensor-arm64-values.yaml"

make_qa_e2e_test_runner(cluster=GKECluster("qa-e2e-test", machine_type="t2a-standard-4")).run()
