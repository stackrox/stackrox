#!/usr/bin/env -S python3 -u

"""
Run QA e2e tests against a given cluster.
"""
import os
from pre_tests import PreSystemTests
from ci_tests import (
    QaE2eTestPart1,
    QaE2eTestPart2,
    QaE2eDBBackupRestoreTest,
    CustomSetTest,
)
from post_tests import PostClusterTest, CheckStackroxLogs, FinalPost
from runners import ClusterTestSetsRunner, TestSet


class QaE2eTestRunner(ClusterTestSetsRunner):
    def run(self):
        # This test suite has been migrated to use roxie for deployment (deploy_stackrox_with_roxie_compat()) instead of
        # the legacy deployment flow (deploy_stackrox()).
        #
        # The previous deployment mechanism used environment variables extensively for deployment configuration.
        # These variables were injected into deployment manifests and/or translated into roxctl command-line arguments
        # in multiple places, which makes the whole configuration setup difficult to maintain and reason about.
        #
        # The compatibility layer for roxie-based deployments (deploy_stackrox_with_roxie_compat()) is designed as a
        # drop-in replacement for the legacy deployment mechanism (deploy_stackrox()) and picks up the same environment
        # variables for configuration with the same defaulting behaviour.
        #
        # Long term, the goal is to migrate all test suites to use the modern roxie-based deployment mechanism,
        # where the entire deployment configuration is to be assembled explicitly in a YAML configuration file.
        os.environ.setdefault("USE_ROXIE_DEPLOY", "true")
        super().run()


def make_qa_e2e_test_runner(cluster):
    return QaE2eTestRunner(
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
    return QaE2eTestRunner(
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
