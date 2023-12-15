#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in an OCP cluster.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
