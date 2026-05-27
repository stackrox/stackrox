#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a IBM CLOUD Z Openshift cluster provided via
automation-flavors/ibmcloudz.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner_custom
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["LOAD_BALANCER"] = "route"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["KUBERNETES_PROVIDER"] = "ocp"
os.environ["USE_MIDSTREAM_IMAGES"] = "true"
os.environ["REMOTE_CLUSTER_ARCH"] = "s390x"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# IBM Cloud Z (s390x) specific timeouts - ROX-21457 mitigation
# s390x infrastructure has slower provisioning times than x86_64
# Extended timeouts address K8S_API_TIMEOUT and BOOTSTRAP_TIMEOUT failures
os.environ["OPENSHIFT_INSTALL_BOOTSTRAP_TIMEOUT"] = "90m"  # default is 40m
os.environ["OPENSHIFT_INSTALL_API_WAIT_TIMEOUT"] = "45m"   # default is 30m
os.environ["OPERATOR_TIMEOUT"] = "20m"  # Wait for operators to become available

make_qa_e2e_test_runner_custom(cluster=AutomationFlavorsCluster()).run()
