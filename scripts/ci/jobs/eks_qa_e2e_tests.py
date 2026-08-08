#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in an EKS cluster provided via automation-flavors/eks.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["KUBERNETES_PROVIDER"] = "eks"

os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# EKS clusters lack the EBS CSI driver addon, so PVCs can't be provisioned.
# Use emptyDir for scanner-v4-db since persistence isn't needed in CI.
os.environ["SCANNER_V4_DB_PERSISTENCE_NONE"] = "true"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
