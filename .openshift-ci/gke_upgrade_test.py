#!/usr/bin/env -S python3 -u

"""
Run the upgrade test in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import UpgradeTest
from post_tests import PostClusterTest

os.environ["LOAD_BALANCER"] = "lb"
ClusterTestRunner(
    cluster=GKECluster("upgrade-test"),
    pre_test=PreSystemTests(),
    test=UpgradeTest(),
    post_test=PostClusterTest(),
).run()
