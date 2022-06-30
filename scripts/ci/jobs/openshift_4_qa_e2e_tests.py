#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a openshift 4 cluster provided via a hive cluster_claim.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import NullCluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["OPENSHIFT_CI_CLUSTER_CLAIM"] = "openshift-4"

make_qa_e2e_test_runner(cluster=NullCluster()).run()
