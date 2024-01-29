#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a IBM CLOUD POWERVS Openshift cluster provided via
automation-flavors/powervs.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner_custom
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["USE_MIDSTREAM_IMAGES"] = "true"
os.environ["REMOTE_CLUSTER_ARCH"] = "ppc64le"
os.environ["COLLECTION_METHOD"] = "ebpf"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# Trigger tests
make_qa_e2e_test_runner_custom(cluster=AutomationFlavorsCluster()).run()
