#!/usr/bin/env -S python3 -u

"""
Run the Scanner V4 installation tests in an OCP cluster
"""
import os
import re
from runners import ClusterTestRunner
from clusters import AutomationFlavorsCluster
from ci_tests import ScannerV4InstallTest
from pre_tests import PreSystemTests
from post_tests import PostClusterTest, FinalPost

os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["STORE_METRICS"] = "true"
os.environ["ROX_BASELINE_GENERATION_DURATION"] = "5m"
os.environ["ROX_SCANNER_V4"] = "true"
os.environ["ENABLE_OPERATOR_TESTS"] = "true"

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

ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=PreSystemTests(),
    test=ScannerV4InstallTest(),
    post_test=PostClusterTest(
        # StackRox is torn down as part of each test execution so data
        # collection and standard log checks are skipped in this post suite
        # step. The scanner-v4-install test teardown() handles debug collection.
        collect_collector_metrics=False,
        collect_central_artifacts=False,
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(),
).run()
