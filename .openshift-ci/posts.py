#!/usr/bin/env python3

"""
Common steps for after tests are complete
"""

import subprocess
from typing import List


class NullPost:
    def run(self):
        pass


class PostClusterTest:
    API_TIMEOUT = 5 * 60
    COLLECT_TIMEOUT = 5 * 60

    def __init__(self):
        self.exitstatus = 0
        self.failed_commands: List[List[str]] = []
        self.k8s_namespaces = ["stackrox", "stackrox-operator", "proxies", "squid"]
        self.openshift_namespaces = [
            "openshift-dns",
            "openshift-apiserver",
            "openshift-authentication",
            "openshift-etcd",
            "openshift-controller-manager",
        ]

    def run(self):
        self.wait_for_central_api()
        self.collect_cluster_api_data()
        self.collect_infrastructure_logs()
        self.collect_collector_metrics()
        self.get_central_debug_dump()
        self.get_central_diagnostics()
        if self.exitstatus != 0:
            for args in self.failed_commands:
                print(f"Post failure in: {args}")
            raise RuntimeError(f"Post failed: exit {self.exitstatus}")

    def wait_for_central_api(self):
        self._run_with_best_effort(
            ["tests/e2e/lib.sh", "wait_for_api"],
            timeout=PostClusterTest.API_TIMEOUT,
        )

    def collect_cluster_api_data(self):
        for namespace in self.k8s_namespaces + self.openshift_namespaces:
            self._run_with_best_effort(
                ["scripts/ci/collect-service-logs.sh", namespace],
                timeout=PostClusterTest.COLLECT_TIMEOUT,
            )

    def collect_infrastructure_logs(self):
        self._run_with_best_effort(
            ["scripts/ci/collect-infrastructure-logs.sh"],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )

    def collect_collector_metrics(self):
        self._run_with_best_effort(
            ["scripts/ci/collect-collector-metrics.sh", "stackrox"],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )

    def get_central_debug_dump(self):
        self._run_with_best_effort(
            ["scripts/ci/lib.sh", "get_central_debug_dump", "debug-dump"],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )

    def get_central_diagnostics(self):
        self._run_with_best_effort(
            ["scripts/ci/lib.sh", "get_central_diagnostics", "diagnostic-bundle"],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )

    def _run_with_best_effort(self, args: List[str], timeout: int):
        try:
            subprocess.run(
                args,
                check=True,
                timeout=timeout,
            )
        except Exception as err:
            print(f"Exception raised in {args}, {err}")
            self.failed_commands.append(args)
            self.exitstatus = 1
