#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import GKECluster

# set test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["GCP_IMAGE_TYPE"] = "cos_containerd"

os.environ["ROX_ACTIVE_VULN_MGMT"] = "true"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# tbd: restore this file and create gke_qa_e2e_arm64_tests.py with below config
os.environ["REMOTE_CLUSTER_ARCH"] = "arm64"
os.environ["ARM64_NODESELECTORS"] = "true"
os.environ["IMAGE_PREFETCH_DISABLED"] = "true"

make_qa_e2e_test_runner(cluster=GKECluster("qa-e2e-test", machine_type="t2a-standard-8")).run()
