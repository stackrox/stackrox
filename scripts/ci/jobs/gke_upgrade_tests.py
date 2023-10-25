#!/usr/bin/env -S python3 -u

"""
Run the upgrade test in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import UpgradeTest
from post_tests import PostClusterTest, FinalPost

# NOTE:  This test starts with Postgres off so that migrations
# from RocksDB to Postgres can be executed.  Once RocksDB is
# out of support those bits can be removed.

ClusterTestRunner(
    cluster=GKECluster("upgrade-test", machine_type="e2-standard-8", disk_gb=1600),
    pre_test=PreSystemTests(),
    test=UpgradeTest(),
    post_test=PostClusterTest(),
    final_post=FinalPost(
        store_qa_tests_data=True,
    ),
).run()
