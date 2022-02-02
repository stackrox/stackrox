#!/usr/bin/env python3

"""
A hook for OpenShift CI to execute an upgrade test
"""

import subprocess
import sys

class Constants:
    PROVISION_TIMEOUT = 20*60
    WAIT_TIMEOUT = 20*60
    TEST_TIMEOUT = 60*60
    TEARDOWN_TIMEOUT = 5*60
    CLUSTER_ID = "upgrade-test"

class UpgradeTest:
    def __init__(self):
        self.needs_teardown = False
        self.needs_post_analysis = False

    def run(self):
        print("Executing the OpenShift CI upgrade test hook")

        outcome = 0
        try:
            self.provision()
            self.wait()
            self.run_test()
        # pylint: disable=broad-except
        except Exception as err:
            print(f"Exception raised {err}")
            outcome = 1

        if self.needs_post_analysis:
            try:
                self.post_test_analysis()
            # pylint: disable=broad-except
            except Exception as err:
                print(f"Exception raised {err}")
                outcome = 1

        if self.needs_teardown:
            try:
                self.teardown()
            # pylint: disable=broad-except
            except Exception as err:
                print(f"Exception raised {err}")
                outcome = 1

        sys.exit(outcome)

    def provision(self):
        cmd = subprocess.Popen(["scripts/ci/gke.sh", "provision_gke_cluster", Constants.CLUSTER_ID])

        self.needs_teardown = True

        if cmd.wait(Constants.PROVISION_TIMEOUT) != 0:
            raise RuntimeError("Cluster provision failed")

    def wait(self):
        cmd = subprocess.Popen(["scripts/ci/gke.sh", "wait_for_cluster"])

        if cmd.wait(Constants.WAIT_TIMEOUT) != 0:
            raise RuntimeError("Wait for cluster failed")

    def run_test(self):
        # cmd = subprocess.Popen(["tests/upgrade/run.sh"])
        cmd = subprocess.Popen(["sleep", "60"])

        self.needs_post_analysis = True

        if cmd.wait(Constants.TEST_TIMEOUT) != 0:
            raise RuntimeError("Test failed")

    def post_test_analysis(self):
        print(">>>> GATHER DEBUG <<<<")
        return 0

    def teardown(self):
        cmd = subprocess.Popen(["scripts/ci/gke.sh", "teardown_gke_cluster"])

        if cmd.wait(Constants.TEARDOWN_TIMEOUT) != 0:
            raise RuntimeError("Teardown failed")

if __name__ == "__main__":
    UpgradeTest().run()
