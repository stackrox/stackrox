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

# The Postgres version of this test is run from gke_postgres_upgrade_tests.py
exit(0)

# don't use postgres
os.environ["ROX_POSTGRES_DATASTORE"] = "false"

ClusterTestRunner(
    cluster=GKECluster("upgrade-test"),
    pre_test=PreSystemTests(),
    test=UpgradeTest(),
    post_test=PostClusterTest(),
    final_post=FinalPost(
        store_qa_test_debug_logs=True,
        store_qa_spock_results=True,
    ),
).run()
