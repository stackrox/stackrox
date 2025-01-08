#!/usr/bin/env -S python3 -u

"""
Run the Scanner V4 installation tests in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import ScannerV4InstallTest
from post_tests import PostClusterTest, FinalPost

os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["STORE_METRICS"] = "true"
os.environ["ROX_BASELINE_GENERATION_DURATION"] = "5m"
os.environ["ROX_SCANNER_V4"] = "true"

ClusterTestRunner(
    cluster=GKECluster("scanner-v4-install-test", machine_type="e2-standard-8"),
    pre_test=PreSystemTests(),
    test=ScannerV4InstallTest(),
    post_test=PostClusterTest(
        # Stackrox is torn down as part of each test execution so data
        # collection and standard log checks are skipped in this post suite
        # step. The scanner-v4-install test teardown() handles debug collection.
        collect_collector_metrics=False,
        collect_central_artifacts=False,
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(),
).run()
