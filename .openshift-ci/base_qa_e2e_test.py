#!/usr/bin/env -S python3 -u

"""
Run QA e2e tests against a given cluster.
"""
from pre_tests import PreSystemTests
from ci_tests import (
    QaE2eTestPart1,
    QaE2eTestPart2,
    QaE2eDBBackupRestoreTest,
    CustomSetTest,
)
from post_tests import NullPostTest, PostClusterTest, CheckStackroxLogs, FinalPost
from runners import ClusterTestSetsRunner


def make_qa_e2e_test_runner(cluster, post_collect=True):
    return ClusterTestSetsRunner(
        cluster=cluster,
        initial_pre_test=PreSystemTests(),
        sets=[
            {
                "name": "QA tests part I",
                "test": QaE2eTestPart1(),
                "post_test": PostClusterTest(
                    check_stackrox_logs=True,
                    artifact_destination_prefix="part-1",
                ) if post_collect else NullPostTest(),
            },
            {
                "name": "QA tests part II",
                "test": QaE2eTestPart2(),
                "post_test": PostClusterTest(
                    check_stackrox_logs=True,
                    artifact_destination_prefix="part-2",
                ) if post_collect else NullPostTest(),
                "always_run": False,
            },
            {
                "name": "DB backup and restore",
                "test": QaE2eDBBackupRestoreTest(),
                "post_test": CheckStackroxLogs(
                    check_for_errors_in_stackrox_logs=True,
                    artifact_destination_prefix="db-test",
                ) if post_collect else NullPostTest(),
                "always_run": False,
            },
        ],
        final_post=FinalPost(
            store_qa_tests_data=True,
        ),
    )


def make_qa_e2e_test_runner_custom(cluster):
    return ClusterTestSetsRunner(
        cluster=cluster,
        initial_pre_test=PreSystemTests(run_poll_for_system_test_images=False),
        sets=[
            {
                "name": "Custom set of tests for p/z",
                "test": CustomSetTest(),
                "post_test": PostClusterTest(
                    check_stackrox_logs=True,
                    artifact_destination_prefix="custom-pz",
                ),
            },
        ],
        final_post=FinalPost(
            store_qa_tests_data=True,
            handle_e2e_progress_failures=False,
        ),
    )
