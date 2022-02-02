#!/usr/bin/env python3

"""
A hook for OpenShift CI to execute an upgrade test
"""

import subprocess
import sys

class Constants:
    PROVISION_TIMEOUT = 20*60
    WAIT_TIMEOUT = 20*60
    TEARDOWN_TIMEOUT = 20*60
    CLUSTER_ID = "upgrade-test"

def run():
    print("Executing the OpenShift CI upgrade test hook")

    provision()

    wait()

    outcome = 0
    try:
        outcome = run_test()
    # pylint: disable=broad-except
    except Exception as err:
        print(f"test run failed: {err}")
        outcome = 1

    try:
        gather_debug()
    # pylint: disable=broad-except
    except Exception as err:
        print(f"gather debug failed: {err}")
        outcome = 1

    teardown()

    sys.exit(outcome)

def provision():
    try:
        cmd = subprocess.Popen(
            ["scripts/ci/gke.sh", "provision_gke_cluster", Constants.CLUSTER_ID])
    # pylint: disable=broad-except
    except Exception as err:
        # immediate exit - no need to teardown or debug further
        print(f"provision failed: {err}")
        sys.exit(1)

    try:
        cmd.wait(Constants.PROVISION_TIMEOUT)
    except subprocess.TimeoutExpired:
        print("provision timed out")
        teardown()
        sys.exit(1)

    if cmd.returncode != 0:
        print(f"non zero exit from provision: {cmd.returncode}")
        teardown()
        sys.exit(1)

def wait():
    try:
        cmd = subprocess.Popen(
            ["scripts/ci/gke.sh", "wait_for_cluster"])
    # pylint: disable=broad-except
    except Exception as err:
        print(f"wait failed: {err}")
        teardown()
        sys.exit(1)

    try:
        cmd.wait(Constants.WAIT_TIMEOUT)
    except subprocess.TimeoutExpired:
        print("wait timed out")
        teardown()
        sys.exit(1)

    if cmd.returncode != 0:
        print(f"non zero exit from wait: {cmd.returncode}")
        teardown()
        sys.exit(1)

def run_test():
    print(">>>> RUNTEST <<<<")
    return 0

def gather_debug():
    print(">>>> GATHER DEBUG <<<<")
    return 0

def teardown():
    returncode = subprocess.Popen(
        ["scripts/ci/gke.sh", "teardown_gke_cluster"]).wait(Constants.TEARDOWN_TIMEOUT)
    if returncode != 0:
        print(f"non zero exit from teardown: {returncode}")
        sys.exit(1)

if __name__ == "__main__":
    run()
