#!/usr/bin/env -S python3 -u

"""
Run the Scanner V4 tests in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import ScannerV4Test
from post_tests import NullPostTest, FinalPost

os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["STORE_METRICS"] = "true"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_BASELINE_GENERATION_DURATION"] = "5m"
os.environ["ROX_SCANNER_V4"] = "true"

# The current Scanner v4 tests only test Scanner v4 installation in different scenarios.
# Due to the way the tests are structured the different test cases include a teardown at the end.
# This would cause the standard PostClusterTest would to fail, therefore we use the
# NullPostTest() here.
#
# A seperate test focusing on Scanner v4 functionality (as opposed to just installation)
# should use the standard PostClusterTest() machinery.
ClusterTestRunner(
    cluster=GKECluster("scanner-v4-test", machine_type="e2-standard-8"),
    pre_test=PreSystemTests(),
    test=ScannerV4Test(),
    post_test=NullPostTest(),
    final_post=FinalPost(),
).run()
