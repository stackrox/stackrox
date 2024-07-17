#!/usr/bin/env -S python3 -u

"""
Run the Scanner V4 tests in an OCP cluster
"""
import os
from runners import ClusterTestRunner
from clusters import AutomationFlavorsCluster
from ci_tests import ScannerV4Test
from pre_tests import PreSystemTests
from post_tests import PostClusterTest, FinalPost

os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["STORE_METRICS"] = "true"
os.environ["ROX_BASELINE_GENERATION_DURATION"] = "5m"
os.environ["ROX_SCANNER_V4"] = "true"
os.environ["ENABLE_OPERATOR_TESTS"] = "true"

ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=PreSystemTests(),
    test=ScannerV4Test(),
    post_test=PostClusterTest(
        # Stackrox is torn down as part of each test execution so data
        # collection and standard log checks are skipped in this post suite
        # step. The scanner-v4 test teardown() handles debug collection.
        collect_collector_metrics=False,
        collect_central_artifacts=False,
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(),
).run()
