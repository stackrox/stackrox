#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import GKECluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# don't use postgres
os.environ["ROX_POSTGRES_DATASTORE"] = "false"

make_qa_e2e_test_runner(cluster=GKECluster("qa-e2e-test")).run()
