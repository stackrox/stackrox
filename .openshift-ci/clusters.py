#!/usr/bin/env python3

"""
Clusters used in test
"""

import os
import subprocess
import time

from common import popen_graceful_kill


class NullCluster:
    def provision(self):
        pass

    def teardown(self):
        pass


class GKECluster:
    PROVISION_TIMEOUT = 20 * 60
    WAIT_TIMEOUT = 20 * 60
    TEARDOWN_TIMEOUT = 5 * 60
    PROVISION_PATH = "scripts/ci/gke.sh"
    WAIT_PATH = "scripts/ci/gke.sh"
    REFRESH_PATH = "scripts/ci/gke.sh"
    TEARDOWN_PATH = "scripts/ci/gke.sh"

    def __init__(self, cluster_id):
        self.cluster_id = cluster_id
        self.refresh_token_cmd = None

    def provision(self):
        with subprocess.Popen(
            [GKECluster.PROVISION_PATH, "provision_gke_cluster", self.cluster_id]
        ) as cmd:

            try:
                exitstatus = cmd.wait(GKECluster.PROVISION_TIMEOUT)
                if exitstatus != 0:
                    raise RuntimeError(f"Cluster provision failed: exit {exitstatus}")
            except subprocess.TimeoutExpired as err:
                popen_graceful_kill(cmd)
                raise err

        subprocess.run(
            [GKECluster.WAIT_PATH, "wait_for_cluster"],
            check=True,
            timeout=GKECluster.WAIT_TIMEOUT,
        )

        # pylint: disable=consider-using-with
        self.refresh_token_cmd = subprocess.Popen(
            [GKECluster.REFRESH_PATH, "refresh_gke_token"]
        )

        return self

    def teardown(self):
        while os.path.exists("/tmp/hold-cluster"):
            print("Pausing teardown because /tmp/hold-cluster exists")
            time.sleep(60)

        try:
            popen_graceful_kill(self.refresh_token_cmd)
        except Exception as err:
            print(f"Could not terminate the token refresh: {err}")

        subprocess.run(
            [GKECluster.TEARDOWN_PATH, "teardown_gke_cluster"],
            check=True,
            timeout=GKECluster.TEARDOWN_TIMEOUT,
        )

        return self
