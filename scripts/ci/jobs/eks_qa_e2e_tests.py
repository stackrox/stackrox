#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in an EKS cluster provided via automation-flavors/eks.
"""
import os
import tempfile
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["KUBERNETES_PROVIDER"] = "eks"

os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# Use emptyDir for scanner-v4-db since EKS clusters lack a PV provisioner
# (no EBS CSI driver installed). For CI e2e tests, persistence isn't needed.
values_file = tempfile.NamedTemporaryFile(
    mode='w', suffix='.yaml', delete=False, prefix='eks-scanner-v4-'
)
values_file.write('scannerV4:\n  db:\n    persistence:\n      none: true\n')
values_file.close()
os.environ["ROX_CENTRAL_EXTRA_HELM_VALUES_FILE"] = values_file.name

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
