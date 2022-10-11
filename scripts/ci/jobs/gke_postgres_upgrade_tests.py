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

# Start with Postgres off so we can begin with RocksDB and upgrade to Postgres
os.environ["ROX_POSTGRES_DATASTORE"] = "false"

ClusterTestRunner(
    cluster=GKECluster("upgrade-test"),
    pre_test=PreSystemTests(),
    test=PostgresUpgradeTest(),
    post_test=PostClusterTest(),
    final_post=FinalPost(
        store_qa_test_debug_logs=True,
        store_qa_spock_results=True,
    ),
).run()
