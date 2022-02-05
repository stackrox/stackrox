#!/usr/bin/env python3

"""
A hook for OpenShift CI to execute an upgrade test
"""

import subprocess
import sys


class UpgradeTest:
    PROVISION_TIMEOUT = 20 * 60
    WAIT_TIMEOUT = 20 * 60
    TEST_TIMEOUT = 60 * 60
    TEARDOWN_TIMEOUT = 5 * 60
    CLUSTER_ID = "upgrade-test"

    def __init__(self):
        self.needs_teardown = False
        self.needs_post_analysis = False

    def run(self):
        print("Executing the OpenShift CI upgrade test hook")

        exitstatus = 0
        try:
            self.provision()
            self.wait()
            self.run_test()
        # pylint: disable=broad-except
        except Exception as err:
            print(f"Exception raised {err}")
            exitstatus = 1

        if self.needs_post_analysis:
            try:
                self.post_test_analysis()
            # pylint: disable=broad-except
            except Exception as err:
                print(f"Exception raised {err}")
                exitstatus = 1

        if self.needs_teardown:
            try:
                self.teardown()
            # pylint: disable=broad-except
            except Exception as err:
                print(f"Exception raised {err}")
                exitstatus = 1

        sys.exit(exitstatus)

    def provision(self):
        with subprocess.Popen(
            ["scripts/ci/gke.sh", "provision_gke_cluster", UpgradeTest.CLUSTER_ID]
        ) as cmd:

            self.needs_teardown = True
            try:
                exitstatus = cmd.wait(UpgradeTest.PROVISION_TIMEOUT)
                if exitstatus != 0:
                    raise RuntimeError(f"Cluster provision failed: exit {exitstatus}")
            except subprocess.TimeoutExpired as err:
                cmd.kill()
                raise err

    def wait(self):
        subprocess.run(
            ["scripts/ci/gke.sh", "wait_for_cluster"],
            check=True,
            timeout=UpgradeTest.WAIT_TIMEOUT,
        )

    def run_test(self):
        with subprocess.Popen(["tests/upgrade/run.sh"]) as cmd:

            self.needs_post_analysis = True
            try:
                exitstatus = cmd.wait(UpgradeTest.TEST_TIMEOUT)
                if exitstatus != 0:
                    raise RuntimeError(f"Test failed: exit {exitstatus}")
            except subprocess.TimeoutExpired as err:
                cmd.kill()
                raise err

    def post_test_analysis(self):
        print("The future home for debug gathering and analysis")
        return 0

    def teardown(self):
        subprocess.run(
            ["scripts/ci/gke.sh", "teardown_gke_cluster"],
            check=True,
            timeout=UpgradeTest.TEARDOWN_TIMEOUT,
        )


if __name__ == "__main__":
    UpgradeTest().run()
