#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in an AKS cluster provided via automation-flavors/aks.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
