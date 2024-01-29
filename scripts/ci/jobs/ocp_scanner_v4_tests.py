#!/usr/bin/env -S python3 -u

"""
Run the Scanner V4 tests in an OCP cluster
"""
import os
from runners import ClusterTestRunner
from clusters import AutomationFlavorsCluster
from ci_tests import ScannerV4Test
from pre_tests import PreSystemTests
from post_tests import NullPostTest, FinalPost

os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["STORE_METRICS"] = "true"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_BASELINE_GENERATION_DURATION"] = "5m"

# ClusterTestRunner(
    # cluster=AutomationFlavorsCluster(),
    # pre_test=PreSystemTests(),
    # test=ScannerV4Test(),
    # post_test=NullPostTest(),
    # final_post=FinalPost(),
# ).run()
