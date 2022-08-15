#!/usr/bin/env python3

"""
Clusters used in test
"""

import os
import signal
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
    GKE_SCRIPT = "scripts/ci/gke.sh"

    def __init__(self, cluster_id, num_nodes=3, machine_type="e2-standard-4"):
        self.cluster_id = cluster_id
        self.num_nodes = num_nodes
        self.machine_type = machine_type
        self.refresh_token_cmd = None

    def provision(self):
        with subprocess.Popen(
            [
                GKECluster.GKE_SCRIPT,
                "provision_gke_cluster",
                self.cluster_id,
                self.num_nodes,
                self.machine_type,
            ]
        ) as cmd:

            try:
                exitstatus = cmd.wait(GKECluster.PROVISION_TIMEOUT)
                if exitstatus != 0:
                    raise RuntimeError(f"Cluster provision failed: exit {exitstatus}")
            except subprocess.TimeoutExpired as err:
                popen_graceful_kill(cmd)
                raise err

        # OpenShift CI sends a SIGINT when tests are canceled
        signal.signal(signal.SIGINT, self.sigint_handler)

        subprocess.run(
            [GKECluster.GKE_SCRIPT, "wait_for_cluster"],
            check=True,
            timeout=GKECluster.WAIT_TIMEOUT,
        )

        # pylint: disable=consider-using-with
        self.refresh_token_cmd = subprocess.Popen(
            [GKECluster.GKE_SCRIPT, "refresh_gke_token"]
        )

        return self

    def teardown(self):
        while os.path.exists("/tmp/hold-cluster"):
            print("Pausing teardown because /tmp/hold-cluster exists")
            time.sleep(60)

        if self.refresh_token_cmd is not None:
            try:
                popen_graceful_kill(self.refresh_token_cmd)
            except Exception as err:
                print(f"Could not terminate the token refresh: {err}")

        subprocess.run(
            [GKECluster.GKE_SCRIPT, "teardown_gke_cluster"],
            check=True,
            timeout=GKECluster.TEARDOWN_TIMEOUT,
        )

        return self

    def sigint_handler(self, signum, frame):
        print("Tearing down the cluster due to SIGINT", signum, frame)
        self.teardown()


class AutomationFlavorsCluster:
    KUBECTL_TIMEOUT = 5 * 60

    def provision(self):
        kubeconfig = os.environ["KUBECONFIG"]

        print(f"Using kubeconfig from {kubeconfig}")

        print("Nodes:")
        subprocess.run(
            ["kubectl", "get", "nodes", "-o", "wide"],
            check=True,
            timeout=AutomationFlavorsCluster.KUBECTL_TIMEOUT,
        )

        return self

    def teardown(self):
        pass
