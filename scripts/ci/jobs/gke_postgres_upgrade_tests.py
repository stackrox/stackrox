#!/usr/bin/env -S python3 -u

"""
Run the upgrade test in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import PostgresUpgradeTest
from post_tests import PostClusterTest, FinalPost

# NOTE:  This test starts with Postgres off so that migrations
# from RocksDB to Postgres can be executed.  Once RocksDB is
# out of support those bits can be removed.

os.environ["COLLECTION_METHOD"] = "ebpf"

ClusterTestRunner(
    cluster=GKECluster("upgrade-test", machine_type="e2-standard-8"),
    pre_test=PreSystemTests(),
    test=PostgresUpgradeTest(),
    post_test=PostClusterTest(),
    final_post=FinalPost(
        store_qa_test_debug_logs=True,
        store_qa_spock_results=True,
    ),
).run()
