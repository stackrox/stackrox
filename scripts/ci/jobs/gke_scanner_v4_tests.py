#!/usr/bin/env -S python3 -u

"""
Run the Scanner V4 tests in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import ScannerV4Test
from post_tests import PostClusterTest, FinalPost

os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["OUTPUT_FORMAT"] = "helm"
os.environ["ROX_SCANNER_V4_ENABLED"] = "true"
os.environ["STORE_METRICS"] = "true"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_BASELINE_GENERATION_DURATION"] = "5m"

ClusterTestRunner(
    cluster=GKECluster("scanner-v4-test", machine_type="e2-standard-8"),
    pre_test=PreSystemTests(),
    test=ScannerV4Test(),
    post_test=PostClusterTest(
        check_stackrox_logs=True,
    ),
    final_post=FinalPost(),
).run()
