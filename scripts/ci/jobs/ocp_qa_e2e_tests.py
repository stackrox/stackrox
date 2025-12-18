#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in an OCP cluster.
"""
import os
import re
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
# Workload identities are only set up for `openshift-4` infra clusters.
if 'openshift-4' in os.environ.get('CLUSTER_FLAVOR_VARIANT', ''):
    os.environ["SETUP_WORKLOAD_IDENTITIES"] = "true"

# ROX-32314, move out
try:
    # SFA Agent supports OCP starting from 4.16, since we test oldest (4.12) and
    # latest (4.20 at the moment), exclude the former one.
    # We expect CLUSTER_FLAVOR_VARIANT be the following format:
    #   openshift-4-ocp/stable-${major_version}.${minor_version}
    ocp_variant = os.environ.get('CLUSTER_FLAVOR_VARIANT', '')
    EXPR = r"openshift-4-ocp/\w+-(?P<major>\d+).(?P<minor>\d+)"

    m = re.match(EXPR, ocp_variant)
    if int(m.group("major")) >= 4 and int(m.group("minor")) >= 16:
        os.environ["SFA_AGENT"] = "Enabled"
except Exception as ex:
    print(f"Could not identify the OCP version, {ex}, SFA is disabled")

os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
