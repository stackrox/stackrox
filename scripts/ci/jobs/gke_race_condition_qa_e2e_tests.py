#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster with a `-race` stackrox/main.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import GKECluster

# set test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["GCP_IMAGE_TYPE"] = "cos_containerd"

# use -rcd image for stackrox/main
os.environ["MAIN_IMAGE_TAG"] = os.environ["STACKROX_BUILD_TAG"] + "-rcd"
os.environ["CENTRAL_DB_IMAGE_TAG"] = os.environ["STACKROX_BUILD_TAG"]
os.environ["USE_LOCAL_ROXCTL"] = "true"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

make_qa_e2e_test_runner(cluster=GKECluster("race-condition-qa-e2e-test")).run()
