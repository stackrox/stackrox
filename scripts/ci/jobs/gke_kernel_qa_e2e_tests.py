#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster with kernel modules
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import GKECluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# use modules
os.environ["COLLECTION_METHOD"] = "kernel-module"

make_qa_e2e_test_runner(cluster=GKECluster("kernel-qa-e2e-test")).run()
