#!/usr/bin/env python3

"""
Available tests
"""

import subprocess

from common import popen_graceful_kill

QA_TESTS_OUTPUT_DIR = "/tmp/qa-tests-backend-logs"

class BaseTest:
    def __init__(self):
        self.test_output_dirs = []


class NullTest(BaseTest):
    def run(self):
        pass


class UpgradeTest(BaseTest):
    TEST_TIMEOUT = 60 * 60
    TEST_OUTPUT_DIR = "/tmp/upgrade-test-logs"

    def run(self):
        print("Executing the Upgrade Test")

        with subprocess.Popen(
            ["tests/upgrade/run.sh", UpgradeTest.TEST_OUTPUT_DIR]
        ) as cmd:

            self.test_output_dirs = [UpgradeTest.TEST_OUTPUT_DIR, QA_TESTS_OUTPUT_DIR]

            try:
                exitstatus = cmd.wait(UpgradeTest.TEST_TIMEOUT)
                if exitstatus != 0:
                    raise RuntimeError(f"Test failed: exit {exitstatus}")
            except subprocess.TimeoutExpired as err:
                popen_graceful_kill(cmd)
                raise err
