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
    # Provisioning timeout is tightly coupled to the time it may take gke.sh to
    # create a cluster.
    PROVISION_TIMEOUT = 140 * 60
    WAIT_TIMEOUT = 20 * 60
    TEARDOWN_TIMEOUT = 5 * 60
    # separate script names used for testability - test_clusters.py
    PROVISION_PATH = "scripts/ci/gke.sh"
    WAIT_PATH = "scripts/ci/gke.sh"
    REFRESH_PATH = "scripts/ci/gke.sh"
    TEARDOWN_PATH = "scripts/ci/gke.sh"

    def __init__(self, cluster_id, num_nodes=None, machine_type=None, disk_gb=None):
        self.cluster_id = cluster_id
        self.num_nodes = num_nodes
        self.machine_type = machine_type
        self.disk_gb = disk_gb
        self.refresh_token_cmd = None

    def provision(self):
        if self.num_nodes is not None:
            os.environ["NUM_NODES"] = str(self.num_nodes)
        if self.machine_type is not None:
            os.environ["MACHINE_TYPE"] = str(self.machine_type)
        if self.disk_gb is not None:
            os.environ["DISK_SIZE_GB"] = str(self.disk_gb)

        with subprocess.Popen(
            [
                GKECluster.PROVISION_PATH,
                "provision_gke_cluster",
                self.cluster_id,
            ]
        ) as cmd:

            try:
                exitstatus = cmd.wait(GKECluster.PROVISION_TIMEOUT)
                if exitstatus != 0:
                    raise RuntimeError(
                        f"Cluster provision failed: exit {exitstatus}")
            except subprocess.TimeoutExpired as err:
                popen_graceful_kill(cmd)
                raise err

        # OpenShift CI sends a SIGINT when tests are canceled
        signal.signal(signal.SIGINT, self.sigint_handler)

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

    def teardown(self, canceled=False):
        while os.path.exists("/tmp/hold-cluster"):
            print("Pausing teardown because /tmp/hold-cluster exists")
            time.sleep(60)

        if self.refresh_token_cmd is not None and not canceled:
            print("Terminating GKE token refresh")
            try:
                popen_graceful_kill(self.refresh_token_cmd)
            except Exception as err:
                print(f"Could not terminate the token refresh: {err}")

        args = [GKECluster.TEARDOWN_PATH, "teardown_gke_cluster"]
        if canceled:
            args.append("true")
        subprocess.run(
            args,
            check=True,
            timeout=GKECluster.TEARDOWN_TIMEOUT,
        )

        return self

    def sigint_handler(self, signum, frame):
        print("Tearing down the cluster due to SIGINT", signum, frame)
        self.teardown(canceled=True)


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
