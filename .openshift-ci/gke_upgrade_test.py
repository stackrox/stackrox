#!/usr/bin/env -S python3 -u

"""
A hook for OpenShift CI to execute an upgrade test in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from ci_tests import UpgradeTest
from posts import PostClusterTest

os.environ["LOAD_BALANCER"] = "lb"
ClusterTestRunner(
    cluster=GKECluster("upgrade-test"), test=UpgradeTest(), post=PostClusterTest()
).run()
