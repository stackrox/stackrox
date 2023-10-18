#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a openshift 4 cluster provided via a hive cluster_claim.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import OpenShiftScaleWorkersCluster

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"

# Scale up the cluster to support postgres
cluster = OpenShiftScaleWorkersCluster(increment=1)

make_qa_e2e_test_runner(cluster=cluster).run()
