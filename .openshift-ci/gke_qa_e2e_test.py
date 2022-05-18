#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster
"""
import os
from pre_tests import PreSystemTests
from ci_tests import QaE2eTestPart1, QaE2eTestPart2, QaE2eDBBackupRestoreTest
from post_tests import PostClusterTest, StoreArtifacts, FinalPost
from clusters import GKECluster
from runners import ClusterTestSetsRunner

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# override default test environment
os.environ["LOAD_BALANCER"] = "lb"

ClusterTestSetsRunner(
    cluster=GKECluster("qa-e2e-test"),
    sets=[
        {
            "name": "QA tests part I",
            "pre_test": PreSystemTests(),
            "test": QaE2eTestPart1(),
            "post_test": PostClusterTest(
                check_stackrox_logs=True,
                store_qa_test_debug_logs=False,
                store_qa_spock_results=False,
                artifact_destination="part-1",
            ),
        },
        {
            "name": "QA tests part II",
            "test": QaE2eTestPart2(),
            "post_test": PostClusterTest(
                check_stackrox_logs=True,
                store_qa_test_debug_logs=True,
                store_qa_spock_results=True,
                artifact_destination="part-2",
            ),
        },
        {
            "name": "DB backup and restore",
            "test": QaE2eDBBackupRestoreTest(),
            "post_test": StoreArtifacts(),
        },
    ],
    final_post=FinalPost(),
).run()
