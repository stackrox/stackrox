#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster with a postgres DB
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import GKECluster

# set test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["GCP_IMAGE_TYPE"] = "cos_containerd"

# use postgres
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_NEW_POLICY_CATEGORIES"] = "true"

make_qa_e2e_test_runner(cluster=GKECluster("postgres-qa-e2e-test")).run()
