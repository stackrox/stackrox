#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import GKECluster

# set test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["KUBERNETES_PROVIDER"] = "gke"
os.environ["USE_ROXIE_DEPLOY"] = "true"
os.environ["GCP_IMAGE_TYPE"] = "cos_containerd"

os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "stackrox-gke-ssd"

os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

make_qa_e2e_test_runner(cluster=GKECluster("qa-e2e-test")).run()
