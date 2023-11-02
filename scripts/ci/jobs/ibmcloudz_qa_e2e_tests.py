#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a IBM CLOUD Z Openshift cluster provided via automation-flavors/ibmcloudz.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner_custom
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["USE_MIDSTREAM_IMAGES"] = "true"
os.environ["COLLECTION_METHOD"] = "ebpf"
os.environ["REMOTE_CLUSTER_ARCH"] = "s390x"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

make_qa_e2e_test_runner_custom(cluster=AutomationFlavorsCluster()).run()
