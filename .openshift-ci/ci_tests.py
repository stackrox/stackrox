#!/usr/bin/env python3

"""
Available tests
"""

import subprocess

from common import popen_cleanup


class NullTest:
    def run(self):
        pass


class UpgradeTest:
    TEST_TIMEOUT = 60 * 60

    def run(self):
        print("Executing the Upgrade Test")

        with subprocess.Popen(["tests/upgrade/run.sh"]) as cmd:

            try:
                exitstatus = cmd.wait(UpgradeTest.TEST_TIMEOUT)
                if exitstatus != 0:
                    raise RuntimeError(f"Test failed: exit {exitstatus}")
            except subprocess.TimeoutExpired as err:
                popen_cleanup(cmd)
                raise err
