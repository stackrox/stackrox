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
from post_tests import PostClusterTest, CheckStackroxLogs, FinalPost
from runners import ClusterTestSetsRunner, TestSet


def make_qa_e2e_test_runner(cluster):
    return ClusterTestSetsRunner(
        cluster=cluster,
        initial_pre_test=PreSystemTests(),
        sets=[
            TestSet(
                "QA tests part I",
                test=QaE2eTestPart1(),
                post=PostClusterTest(
                    check_stackrox_logs=True,
                    artifact_destination_prefix="part-1",
                ),
            ),
            TestSet(
                "QA tests part II",
                test=QaE2eTestPart2(),
                post=PostClusterTest(
                    check_stackrox_logs=True,
                    artifact_destination_prefix="part-2",
                ),
                always_run=False,
            ),
            TestSet(
                "DB backup and restore",
                test=QaE2eDBBackupRestoreTest(),
                post=CheckStackroxLogs(
                    check_for_errors_in_stackrox_logs=True,
                    artifact_destination_prefix="db-test",
                ),
                always_run=False,
            ),
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
            TestSet(
                "Custom set of tests for p/z",
                test=CustomSetTest(),
                post=PostClusterTest(
                    check_stackrox_logs=True,
                    artifact_destination_prefix="custom-pz",
                ),
            ),
        ],
        final_post=FinalPost(
            store_qa_tests_data=True,
            handle_e2e_progress_failures=False,
        ),
    )
