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
os.environ["SETUP_WORKLOAD_IDENTITIES"] = "true"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

for v in ['JOB_NAME_SAFE', 'CLUSTER_FLAVOR_VARIANT', 'CLUSTER_PROFILE_NAME', 'CLUSTER_TYPE', 'OPENSHIFT_CI_STEP_NAME']:
    print(v + ':' + os.environ.get(v, ''))

if 'openshift-4' in os.environ.get('CLUSTER_FLAVOR_VARIANT', ''):
    os.environ["SETUP_WORKLOAD_IDENTITIES"] = "false"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
