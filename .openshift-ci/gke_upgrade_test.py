#!/usr/bin/env python3

"""
A hook for OpenShift CI to execute an upgrade test in a GKE cluster
"""

from runners import ClusterTestRunner
from clusters import GKECluster
from ci_tests import UpgradeTest
from posts import PostClusterTest

ClusterTestRunner(
    cluster=GKECluster("upgrade-test"), test=UpgradeTest(), post=PostClusterTest()
).run()
