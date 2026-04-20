#!/usr/bin/env -S python3 -u

"""
Run the upgrade test in a GKE cluster
"""
import os
import scanner_v4_defaults
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import UpgradeTest
from post_tests import PostClusterTest, FinalPost

os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# Scanner V4
os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "faster"
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = scanner_v4_defaults.VULN_BUNDLE_ALLOWLIST

ClusterTestRunner(
    cluster=GKECluster("upgrade-test", machine_type="e2-standard-8"),
    pre_test=PreSystemTests(),
    test=UpgradeTest(),
    post_test=PostClusterTest(),
    final_post=FinalPost(
        store_qa_tests_data=True,
    ),
).run()
