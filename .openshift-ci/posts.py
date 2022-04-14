#!/usr/bin/env python3

"""
Common steps when tests complete
"""

import subprocess
from typing import List


class NullPost:
    def run(self, test_output_dirs=None):
        pass


class PostClusterTest:
    API_TIMEOUT = 5 * 60
    COLLECT_TIMEOUT = 5 * 60
    STORE_TIMEOUT = 5 * 60
    K8S_LOG_DIR = "/tmp/k8s-service-logs"
    COLLECTOR_METRICS_DIR = "/tmp/collector-metrics"
    DEBUG_OUTPUT = "debug-dump"
    DIAGNOSTIC_OUTPUT = "diagnostic-bundle"
    CENTRAL_DATA_OUTPUT = "central-data"

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

    def run(self, test_output_dirs=None):
        self.wait_for_central_api()
        self.collect_service_logs()
        self.collect_collector_metrics()
        self.get_central_debug_dump()
        self.get_central_diagnostics()
        self.grab_central_data()
        self.store_test_output(test_output_dirs)
        self.fixup_artifacts_content_type()
        self.make_artifacts_help()
        if self.exitstatus != 0:
            for args in self.failed_commands:
                print(f"Post failure in: {args}")
            raise RuntimeError(f"Post failed: exit {self.exitstatus}")

    def wait_for_central_api(self):
        self._run_with_best_effort(
            ["tests/e2e/lib.sh", "wait_for_api"],
            timeout=PostClusterTest.API_TIMEOUT,
        )

    def collect_service_logs(self):
        for namespace in self.k8s_namespaces + self.openshift_namespaces:
            self._run_with_best_effort(
                [
                    "scripts/ci/collect-service-logs.sh",
                    namespace,
                    PostClusterTest.K8S_LOG_DIR,
                ],
                timeout=PostClusterTest.COLLECT_TIMEOUT,
            )
        self._run_with_best_effort(
            ["scripts/ci/collect-infrastructure-logs.sh", PostClusterTest.K8S_LOG_DIR],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )
        self._run_with_best_effort(
            [
                "scripts/ci/store-artifacts.sh",
                "store_artifacts",
                PostClusterTest.K8S_LOG_DIR,
            ],
            timeout=PostClusterTest.STORE_TIMEOUT,
        )

    def collect_collector_metrics(self):
        self._run_with_best_effort(
            [
                "scripts/ci/collect-collector-metrics.sh",
                "stackrox",
                PostClusterTest.COLLECTOR_METRICS_DIR,
            ],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )
        self._run_with_best_effort(
            [
                "scripts/ci/store-artifacts.sh",
                "store_artifacts",
                PostClusterTest.COLLECTOR_METRICS_DIR,
            ],
            timeout=PostClusterTest.STORE_TIMEOUT,
        )

    def get_central_debug_dump(self):
        self._run_with_best_effort(
            [
                "scripts/ci/lib.sh",
                "get_central_debug_dump",
                PostClusterTest.DEBUG_OUTPUT,
            ],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )
        self._run_with_best_effort(
            [
                "scripts/ci/store-artifacts.sh",
                "store_artifacts",
                PostClusterTest.DEBUG_OUTPUT,
            ],
            timeout=PostClusterTest.STORE_TIMEOUT,
        )

    def get_central_diagnostics(self):
        self._run_with_best_effort(
            [
                "scripts/ci/lib.sh",
                "get_central_diagnostics",
                PostClusterTest.DIAGNOSTIC_OUTPUT,
            ],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )
        self._run_with_best_effort(
            [
                "scripts/ci/store-artifacts.sh",
                "store_artifacts",
                PostClusterTest.DIAGNOSTIC_OUTPUT,
            ],
            timeout=PostClusterTest.STORE_TIMEOUT,
        )

    def grab_central_data(self):
        self._run_with_best_effort(
            ["scripts/grab-data-from-central.sh", PostClusterTest.CENTRAL_DATA_OUTPUT],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )
        self._run_with_best_effort(
            [
                "scripts/ci/store-artifacts.sh",
                "store_artifacts",
                PostClusterTest.CENTRAL_DATA_OUTPUT,
            ],
            timeout=PostClusterTest.STORE_TIMEOUT,
        )

    def fixup_artifacts_content_type(self):
        self._run_with_best_effort(
            ["scripts/ci/store-artifacts.sh", "fixup_artifacts_content_type"],
            timeout=PostClusterTest.STORE_TIMEOUT,
        )

    def make_artifacts_help(self):
        self._run_with_best_effort(
            ["scripts/ci/store-artifacts.sh", "make_artifacts_help"],
            timeout=PostClusterTest.STORE_TIMEOUT,
        )

    def store_test_output(self, test_output_dirs):
        for output in test_output_dirs:
            self._run_with_best_effort(
                ["scripts/ci/store-artifacts.sh", "store_artifacts", output],
                timeout=PostClusterTest.STORE_TIMEOUT,
            )

    def _run_with_best_effort(self, args: List[str], timeout: int):
        print(f"Running post command: {args}")
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
