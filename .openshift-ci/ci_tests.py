#!/usr/bin/env python3

"""
Available tests
"""

import subprocess

from common import popen_graceful_kill


class BaseTest:
    def __init__(self):
        self.test_outputs = []
        self.test_results = {}

    def run_with_graceful_kill(self, args, timeout, post_start_hook=None):
        with subprocess.Popen(args) as cmd:
            if post_start_hook is not None:
                post_start_hook()
            try:
                exitstatus = cmd.wait(timeout)
                if exitstatus != 0:
                    raise RuntimeError(f"Test failed: exit {exitstatus}")
            except subprocess.TimeoutExpired as err:
                # Kill child processes as we cannot rely on bash scripts to handle signals and stop tests
                subprocess.run(["/usr/bin/pkill", "-P", str(cmd.pid)], check=True, timeout=5)
                # Then kill the test command
                popen_graceful_kill(cmd)
                raise err


class NullTest(BaseTest):
    def run(self):
        pass


class UpgradeTest(BaseTest):
    TEST_TIMEOUT = 60 * 60
    TEST_OUTPUT_DIR = "/tmp/upgrade-test-logs"

    def run(self):
        print("Executing the Upgrade Test")

        def set_dirs_after_start():
            # let post test know where logs are
            self.test_outputs = [UpgradeTest.TEST_OUTPUT_DIR]

        self.run_with_graceful_kill(
            ["tests/upgrade/run.sh", UpgradeTest.TEST_OUTPUT_DIR],
            UpgradeTest.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )


class PostgresUpgradeTest(BaseTest):
    TEST_TIMEOUT = 60 * 60 * 2
    TEST_OUTPUT_DIR = "/tmp/postgres-upgrade-test-logs"
    TEST_LEGACY_OUTPUT_DIR = "/tmp/legacy-postgres-upgrade-test-logs"
    TEST_SENSOR_OUTPUT_DIR = "/tmp/postgres-sensor-upgrade-test-logs"

    def run(self):
        print("Executing the Postgres Upgrade Test")

        def set_dirs_after_start():
            # let post test know where logs are
            self.test_outputs = [PostgresUpgradeTest.TEST_SENSOR_OUTPUT_DIR, PostgresUpgradeTest.TEST_OUTPUT_DIR]

        self.run_with_graceful_kill(
            ["tests/upgrade/postgres_sensor_run.sh", PostgresUpgradeTest.TEST_SENSOR_OUTPUT_DIR],
            PostgresUpgradeTest.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )

        self.run_with_graceful_kill(
            ["tests/upgrade/legacy_to_postgres_run.sh", PostgresUpgradeTest.TEST_LEGACY_OUTPUT_DIR],
            PostgresUpgradeTest.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )

        self.run_with_graceful_kill(
            ["tests/upgrade/postgres_run.sh", PostgresUpgradeTest.TEST_OUTPUT_DIR],
            PostgresUpgradeTest.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )


class OperatorE2eTest(BaseTest):
    # TODO(ROX-12348): adjust these timeouts once we know average run times
    FETCH_KUTTL_TIMEOUT_SEC = 5 * 60
    DEPLOY_TIMEOUT_SEC = 40 * 60
    UPGRADE_TEST_TIMEOUT_SEC = 50 * 60
    E2E_TEST_TIMEOUT_SEC = 50 * 60
    SCORECARD_TEST_TIMEOUT_SEC = 20 * 60

    def __init__(self):
        self.test_outputs = [
            "operator/build/kuttl-test-artifacts",
            "operator/build/kuttl-test-artifacts-upgrade",
        ]
        self.test_results = {
            "kuttl-test-artifacts": "operator/build/kuttl-test-artifacts",
            "kuttl-test-artifacts-upgrade": "operator/build/kuttl-test-artifacts-upgrade",
        }

    def run(self):
        print("Fetching kuttl binary")
        self.run_with_graceful_kill(
            ["operator/hack/junit_wrap.sh", "fetch-kuttl",
             "Download kuttl binary.", "See log for error details.",
             "make", "-C", "operator", "kuttl"],
            OperatorE2eTest.FETCH_KUTTL_TIMEOUT_SEC,
        )
        print("Deploying operator")
        self.run_with_graceful_kill(
            ["operator/hack/junit_wrap.sh", "deploy-previous-operator",
             "Deploy previously released version of the operator.",
             "See log for error details. Reading operator/tests/TROUBLESHOOTING_E2E_TESTS.md may also be helpful.",
             "make", "-C", "operator", "deploy-previous-via-olm"],
            OperatorE2eTest.DEPLOY_TIMEOUT_SEC,
        )

        print("Executing operator upgrade test")
        self.run_with_graceful_kill(
            ["operator/hack/junit_wrap.sh", "test-upgrade",
             "Test operator upgrade from previously released version to the current one.",
             "See log and/or kuttl JUnit output for error details. "
             "Reading operator/tests/TROUBLESHOOTING_E2E_TESTS.md may also be helpful.",
             "make", "-C", "operator", "test-upgrade"],
            OperatorE2eTest.UPGRADE_TEST_TIMEOUT_SEC,
        )

        print("Executing operator e2e tests")
        self.run_with_graceful_kill(
            ["operator/hack/junit_wrap.sh", "test-e2e",
             "Run operator E2E tests.",
             "See log and/or kuttl JUnit output for error details. "
             "Reading operator/tests/TROUBLESHOOTING_E2E_TESTS.md may also be helpful.",
             "make", "-C", "operator", "test-e2e-deployed"],
            OperatorE2eTest.E2E_TEST_TIMEOUT_SEC,
        )

        print("Executing Operator Bundle Scorecard tests")
        self.run_with_graceful_kill(
            [
                "operator/hack/junit_wrap.sh", "bundle-test-image",
                "Run scorecard tests.",
                "See log for error details.",
                "./operator/scripts/retry.sh",
                "4",
                "2",
                "make",
                "-C",
                "operator",
                "bundle-test-image",
            ],
            OperatorE2eTest.SCORECARD_TEST_TIMEOUT_SEC,
        )


class QaE2eTestPart1(BaseTest):
    TEST_TIMEOUT = 240 * 60

    def run(self):
        print("Executing qa-tests-backend tests (part I)")

        self.run_with_graceful_kill(
            ["qa-tests-backend/scripts/run-part-1.sh"], QaE2eTestPart1.TEST_TIMEOUT
        )


class QaE2eTestPart2(BaseTest):
    TEST_TIMEOUT = 30 * 60

    def run(self):
        print("Executing qa-tests-backend tests (part II)")

        self.run_with_graceful_kill(
            ["qa-tests-backend/scripts/run-part-2.sh"], QaE2eTestPart2.TEST_TIMEOUT
        )

class QaE2eTestCompatibility(BaseTest):
    TEST_TIMEOUT = 240 * 60

    def run(self):
        print("Executing qa-tests-compatibility tests")

        self.run_with_graceful_kill(
            ["qa-tests-backend/scripts/run-compatibility.sh"], QaE2eTestCompatibility.TEST_TIMEOUT
        )

class QaE2eDBBackupRestoreTest(BaseTest):
    TEST_TIMEOUT = 30 * 60
    TEST_OUTPUT_DIR = "/tmp/db-backup-restore-test"

    def run(self):
        print("Executing DB backup and restore test")

        def set_dirs_after_start():
            # let post test know where logs are
            self.test_outputs = [QaE2eDBBackupRestoreTest.TEST_OUTPUT_DIR]

        self.run_with_graceful_kill(
            [
                "tests/e2e/lib.sh",
                "db_backup_and_restore_test",
                QaE2eDBBackupRestoreTest.TEST_OUTPUT_DIR,
            ],
            QaE2eDBBackupRestoreTest.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )


class UIE2eTest(BaseTest):
    TEST_TIMEOUT = 2 * 60 * 60

    def run(self):
        print("Executing UI e2e test")

        self.run_with_graceful_kill(
            [
                "tests/e2e/run-ui-e2e.sh",
            ],
            UIE2eTest.TEST_TIMEOUT,
        )


class NonGroovyE2e(BaseTest):
    TEST_TIMEOUT = 90 * 60
    TEST_OUTPUT_DIR = "/tmp/e2e-test-logs"

    def run(self):
        print("Executing the E2e Test")

        def set_dirs_after_start():
            # let post test know where logs are
            self.test_outputs = [NonGroovyE2e.TEST_OUTPUT_DIR]

        self.run_with_graceful_kill(
            ["tests/e2e/run.sh", NonGroovyE2e.TEST_OUTPUT_DIR],
            NonGroovyE2e.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )


class ScaleTest(BaseTest):
    TEST_TIMEOUT = 90 * 60
    PPROF_ZIP_OUTPUT = "/tmp/scale-test/pprof.zip"

    def run(self):
        print("Executing the Scale Test")

        def set_dirs_after_start():
            # let post test know where results are
            self.test_outputs = [ScaleTest.PPROF_ZIP_OUTPUT]

        self.run_with_graceful_kill(
            ["tests/e2e/run-scale.sh", ScaleTest.PPROF_ZIP_OUTPUT],
            ScaleTest.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )
