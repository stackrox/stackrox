#!/usr/bin/env python3

"""
Available tests
"""

import subprocess

from common import popen_graceful_kill


class BaseTest:
    def __init__(self, *args, **kwargs):
        self.test_outputs = []

    def run_with_graceful_kill(self, args, timeout, post_start_hook=None):
        with subprocess.Popen(args) as cmd:
            if post_start_hook is not None:
                post_start_hook()
            try:
                exitstatus = cmd.wait(timeout * 2)
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
    TEST_TIMEOUT = 60 * 60 * 2
    TEST_OUTPUT_DIR = "/tmp/postgres-upgrade-test-logs"
    TEST_LEGACY_OUTPUT_DIR = "/tmp/legacy-postgres-upgrade-test-logs"
    TEST_SENSOR_OUTPUT_DIR = "/tmp/postgres-sensor-upgrade-test-logs"

    def run(self):
        print("Executing the Upgrade Test")

        def set_dirs_after_start():
            # let post test know where logs are
            self.test_outputs = [UpgradeTest.TEST_SENSOR_OUTPUT_DIR, UpgradeTest.TEST_OUTPUT_DIR]

        self.run_with_graceful_kill(
            ["tests/upgrade/postgres_sensor_run.sh", UpgradeTest.TEST_SENSOR_OUTPUT_DIR],
            UpgradeTest.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )

        self.run_with_graceful_kill(
            ["tests/upgrade/legacy_to_postgres_run.sh", UpgradeTest.TEST_LEGACY_OUTPUT_DIR],
            UpgradeTest.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )

        self.run_with_graceful_kill(
            ["tests/upgrade/postgres_run.sh", UpgradeTest.TEST_OUTPUT_DIR],
            UpgradeTest.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )


class OperatorE2eTest(BaseTest):
    OLM_SETUP_TIMEOUT_SEC = 60 * 10
    TEST_TIMEOUT_SEC = 60 * 60 * 2
    OPERATOR_CLUSTER_TYPE_OPENSHIFT4 = "openshift4"

    def __init__(self, *args, **kwargs):
        super(OperatorE2eTest, self).__init__(*args, **kwargs)
        self._operator_cluster_type = kwargs.get(
            "operator_cluster_type", OperatorE2eTest.OPERATOR_CLUSTER_TYPE_OPENSHIFT4)

    def run(self):
        if self._operator_cluster_type != OperatorE2eTest.OPERATOR_CLUSTER_TYPE_OPENSHIFT4:
            print("Running on cluster type %s, installing OLM" % self._operator_cluster_type)
            self.run_with_graceful_kill(
                ["make", "-C", "operator", "olm-install"],
                OperatorE2eTest.TEST_TIMEOUT_SEC,
            )

        print("Executing operator e2e tests")
        self.run_with_graceful_kill(
            ["operator/tests/run.sh"],
            OperatorE2eTest.TEST_TIMEOUT_SEC,
        )


class QaE2eTestPart1(BaseTest):
    TEST_TIMEOUT = 240 * 60 * 2

    def run(self):
        print("Executing qa-tests-backend tests (part I)")

        self.run_with_graceful_kill(
            ["qa-tests-backend/scripts/run-part-1.sh"], QaE2eTestPart1.TEST_TIMEOUT
        )


class QaE2eTestPart2(BaseTest):
    TEST_TIMEOUT = 30 * 60 * 2

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


class ComplianceE2eTest(BaseTest):
    TEST_TIMEOUT = 2 * 60 * 60

    def run(self):
        print("Executing compliance e2e test")

        self.run_with_graceful_kill(
            [
                "tests/e2e/run-compliance-e2e.sh",
            ],
            ComplianceE2eTest.TEST_TIMEOUT,
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


class SensorIntegration(BaseTest):
    TEST_TIMEOUT = 90 * 60
    TEST_OUTPUT_DIR = "/tmp/sensor-integration-test-logs"

    def run(self):
        print("Executing the Sensor Integration Tests")

        def set_dirs_after_start():
            # let post test know where logs are
            self.test_outputs = [SensorIntegration.TEST_OUTPUT_DIR]

        self.run_with_graceful_kill(
            ["tests/e2e/sensor.sh", SensorIntegration.TEST_OUTPUT_DIR],
            SensorIntegration.TEST_TIMEOUT,
            post_start_hook=set_dirs_after_start,
        )


class SensorIntegrationOCP(SensorIntegration):
    def run(self):
        print("Skipping the Sensor Integration Tests for OCP")



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

class CustomSetTest(BaseTest):
    TEST_TIMEOUT = 240 * 60

    def run(self):
        print("Executing a sub set of qa-tests-backend tests for power and s390x")

        self.run_with_graceful_kill(
            ["qa-tests-backend/scripts/run-custom-pz.sh"], CustomSetTest.TEST_TIMEOUT
        )
