#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster
"""
import os
import scanner_v4_defaults
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import GKECluster

# set test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["GCP_IMAGE_TYPE"] = "cos_containerd"
os.environ["OUTPUT_FORMAT"] = "helm"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# Scanner V4
os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "stackrox-gke-ssd"
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = scanner_v4_defaults.VULN_BUNDLE_ALLOWLIST

make_qa_e2e_test_runner(cluster=GKECluster("qa-e2e-test")).run()
