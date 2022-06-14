#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in the provided cluster
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import NullCluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"

# override default test environment
os.environ["LOAD_BALANCER"] = "lb"

make_qa_e2e_test_runner(cluster=NullCluster()).run()
