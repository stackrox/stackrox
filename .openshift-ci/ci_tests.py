#!/usr/bin/env python3

"""
Available tests
"""

import os
import subprocess

from common import popen_graceful_kill


class BaseTest:
    def __init__(self):
        self.test_outputs = []

    def run_with_graceful_kill(self, args, timeout, output_dir=None):
        output_dir_env = {}
        if output_dir:
            if output_dir not in self.test_outputs:
                self.test_outputs.append(output_dir)
            output_dir_env = {"ROX_CI_OUTPUT_DIR": output_dir}

        with subprocess.Popen(args, env=dict(os.environ, **output_dir_env)) as cmd:
            try:
                exitstatus = cmd.wait(timeout)
                if exitstatus != 0:
                    raise RuntimeError(f"Test failed: exit {exitstatus}")
            except subprocess.TimeoutExpired as err:
                # Kill child processes as we cannot rely on bash scripts to
                # handle signals and stop tests
                subprocess.run(
                    ["/usr/bin/pkill", "-P", str(cmd.pid)], check=True, timeout=5
                )
                # Then kill the test command
                popen_graceful_kill(cmd)
                raise err


class NullTest(BaseTest):
    def run(self):
        pass


class UpgradeTest(BaseTest):
    TEST_TIMEOUT = 60 * 60 * 2
    TEST_OUTPUT_DIR = "/tmp/postgres-upgrade-test-logs"
    TEST_PG_UPGRADE_OUTPUT_DIR = "/tmp/postgres-version-upgrade-test-logs"
    TEST_SENSOR_OUTPUT_DIR = "/tmp/postgres-sensor-upgrade-test-logs"

    def run(self):
        print("Executing the Upgrade Test")

        self.run_with_graceful_kill(
            [
                "tests/upgrade/postgres_sensor_run.sh",
                self.TEST_SENSOR_OUTPUT_DIR,
            ],
            self.TEST_TIMEOUT,
            output_dir=self.TEST_SENSOR_OUTPUT_DIR,
        )

        self.run_with_graceful_kill(
            ["tests/upgrade/postgres_run.sh", self.TEST_OUTPUT_DIR],
            self.TEST_TIMEOUT,
            output_dir=self.TEST_OUTPUT_DIR,
        )

        self.run_with_graceful_kill(
            ["tests/upgrade/postgres_upgrade_run.sh", self.TEST_PG_UPGRADE_OUTPUT_DIR],
            self.TEST_TIMEOUT,
            output_dir=self.TEST_PG_UPGRADE_OUTPUT_DIR,
        )


class OperatorE2eTest(BaseTest):
    OLM_SETUP_TIMEOUT_SEC = 60 * 10
    TEST_TIMEOUT_SEC = 60 * 60 * 2
    OPERATOR_CLUSTER_TYPE_OPENSHIFT4 = "openshift4"

    def __init__(self, operator_cluster_type=OPERATOR_CLUSTER_TYPE_OPENSHIFT4):
        super().__init__()
        self._operator_cluster_type = operator_cluster_type

    def run(self):
        print(f"Running on cluster type {self._operator_cluster_type}")
        if (
            self._operator_cluster_type
            == self.OPERATOR_CLUSTER_TYPE_OPENSHIFT4
        ):
            print("Removing unused catalog sources")
            self.run_with_graceful_kill(
                ["kubectl", "patch", "operatorhub.config.openshift.io", "cluster", "--type=json",
                 "-p", '[{"op": "add", "path": "/spec/disableAllDefaultSources", "value": true}]'],
                self.OLM_SETUP_TIMEOUT_SEC,
            )
            olm_ns = "openshift-operator-lifecycle-manager"
        else:
            print("Installing OLM")
            self.run_with_graceful_kill(
                ["make", "-C", "operator", "olm-install"],
                self.OLM_SETUP_TIMEOUT_SEC,
            )
            print("Removing unused catalog source(s)")
            self.run_with_graceful_kill(
                ["kubectl", "delete", "catalogsource.operators.coreos.com",
                    "--namespace=olm", "--all"],
                self.OLM_SETUP_TIMEOUT_SEC,
            )
            olm_ns = "olm"
        print("Bouncing catalog operator pod to clear its cache")
        self.run_with_graceful_kill(
            ["kubectl", "delete", "pods",
                f"--namespace={olm_ns}", "--selector", "app=catalog-operator", "--now=true"],
            self.OLM_SETUP_TIMEOUT_SEC,
        )

        print("Executing operator e2e tests")
        self.run_with_graceful_kill(
            ["operator/tests/run.sh"],
            self.TEST_TIMEOUT_SEC,
            output_dir="/tmp/operator-e2e-misc-logs",
        )


class QaE2eTestPart1(BaseTest):
    TEST_TIMEOUT = 240 * 60

    def run(self):
        print("Executing qa-tests-backend tests (part I)")

        self.run_with_graceful_kill(
            ["qa-tests-backend/scripts/run-part-1.sh"], self.TEST_TIMEOUT,
            output_dir="/tmp/qa-part1-misc-logs",
        )


class QaE2eTestPart2(BaseTest):
    TEST_TIMEOUT = 30 * 60

    def run(self):
        print("Executing qa-tests-backend tests (part II)")

        self.run_with_graceful_kill(
            ["qa-tests-backend/scripts/run-part-2.sh"], self.TEST_TIMEOUT,
            output_dir="/tmp/qa-part2-misc-logs",
        )


class QaE2eTestCompatibility(BaseTest):
    TEST_TIMEOUT = 240 * 60

    def __init__(self, central_version, sensor_version):
        super().__init__()
        self._central_version = central_version
        self._sensor_version = sensor_version

    def run(self):
        print("Executing qa-tests-compatibility tests")

        self.run_with_graceful_kill(
            ["qa-tests-backend/scripts/run-compatibility.sh",
             self._central_version, self._sensor_version],
            self.TEST_TIMEOUT,
            output_dir="/tmp/qa-compat-misc-logs",
        )


class QaE2eGoCompatibilityTest(BaseTest):
    TEST_TIMEOUT = 240 * 60
    TEST_OUTPUT_DIR = "/tmp/compatibility-test-logs"

    def __init__(self, central_version, sensor_version):
        super().__init__()
        self._central_version = central_version
        self._sensor_version = sensor_version

    def run(self):
        print("Executing non-groovy compatibility tests")

        self.run_with_graceful_kill(
            ["tests/e2e/run-compatibility.sh",
             self._central_version, self._sensor_version],
            self.TEST_TIMEOUT,
            output_dir=self.TEST_OUTPUT_DIR,
        )


class QaE2eDBBackupRestoreTest(BaseTest):
    TEST_TIMEOUT = 30 * 60
    TEST_OUTPUT_DIR = "/tmp/db-backup-restore-test"

    def run(self):
        print("Executing DB backup and restore test")

        self.run_with_graceful_kill(
            [
                "tests/e2e/lib.sh",
                "db_backup_and_restore_test",
                self.TEST_OUTPUT_DIR,
            ],
            self.TEST_TIMEOUT,
            output_dir=self.TEST_OUTPUT_DIR,
        )


class UIE2eTest(BaseTest):
    TEST_TIMEOUT = 2 * 60 * 60

    def run(self):
        print("Executing UI e2e test")

        self.run_with_graceful_kill(
            [
                "tests/e2e/run-ui-e2e.sh",
            ],
            self.TEST_TIMEOUT,
            output_dir="/tmp/ui-e2e-misc-logs",
        )


class ComplianceE2eTest(BaseTest):
    TEST_TIMEOUT = 2 * 60 * 60

    def run(self):
        print("Executing compliance e2e test")

        self.run_with_graceful_kill(
            [
                "tests/e2e/run-compliance-e2e.sh",
            ],
            self.TEST_TIMEOUT,
            output_dir="/tmp/compliance-e2e-misc-logs",
        )


class NonGroovyE2e(BaseTest):
    TEST_TIMEOUT = 90 * 60
    TEST_OUTPUT_DIR = "/tmp/e2e-test-logs"

    def run(self):
        print("Executing the E2e Test")

        self.run_with_graceful_kill(
            ["tests/e2e/run.sh", self.TEST_OUTPUT_DIR],
            self.TEST_TIMEOUT,
            output_dir=self.TEST_OUTPUT_DIR,
        )


class SensorIntegration(BaseTest):
    TEST_TIMEOUT = 90 * 60
    TEST_OUTPUT_DIR = "/tmp/sensor-integration-test-logs"

    def run(self):
        print("Executing the Sensor Integration Tests")

        self.run_with_graceful_kill(
            ["tests/e2e/sensor.sh", self.TEST_OUTPUT_DIR],
            self.TEST_TIMEOUT,
            output_dir=self.TEST_OUTPUT_DIR,
        )


class SensorIntegrationOCP(SensorIntegration):
    def run(self):
        # TODO(ROX-17875): make them work on OCP.
        print("Skipping the Sensor Integration Tests for OCP")


class ScaleTest(BaseTest):
    TEST_TIMEOUT = 90 * 60
    PPROF_ZIP_OUTPUT = "/tmp/scale-test/pprof.zip"

    def run(self):
        print("Executing the Scale Test")

        self.run_with_graceful_kill(
            ["tests/e2e/run-scale.sh", self.PPROF_ZIP_OUTPUT],
            self.TEST_TIMEOUT,
            output_dir=self.PPROF_ZIP_OUTPUT,
        )


class ScannerV4InstallTest(BaseTest):
    TEST_TIMEOUT = 240 * 60
    TEST_OUTPUT_DIR = "/tmp/scanner-v4-logs"

    def run(self):
        print("Executing the Scanner V4 Test")

        self.run_with_graceful_kill(
            ["tests/e2e/run-scanner-v4-install.sh", self.TEST_OUTPUT_DIR],
            self.TEST_TIMEOUT,
            output_dir=self.TEST_OUTPUT_DIR,
        )


class CustomSetTest(BaseTest):
    TEST_TIMEOUT = 7 * 60 * 60

    def run(self):
        print("Executing a sub set of qa-tests-backend tests for ppc64le and s390x")

        self.run_with_graceful_kill(
            ["qa-tests-backend/scripts/run-custom-pz.sh"], self.TEST_TIMEOUT,
            output_dir="/tmp/custom-pz-misc-logs",
        )


class BYODBTest(BaseTest):
    TEST_TIMEOUT = 60 * 60 * 2
    TEST_OUTPUT_DIR = "/tmp/byodb-test-logs"

    def run(self):
        print("Executing the BYODB Test")

        self.run_with_graceful_kill(
            ["tests/byodb/run.sh", self.TEST_OUTPUT_DIR],
            self.TEST_TIMEOUT,
            output_dir=self.TEST_OUTPUT_DIR,
        )
