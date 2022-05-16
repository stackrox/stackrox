#!/usr/bin/env python3

"""
Common steps when tests complete
"""

import subprocess
from typing import List


class NullPostTest:
    def run(self, test_output_dirs=None):
        pass


# pylint: disable=too-many-instance-attributes
class PostClusterTest:
    API_TIMEOUT = 5 * 60
    COLLECT_TIMEOUT = 5 * 60
    CHECK_TIMEOUT = 5 * 60
    STORE_TIMEOUT = 5 * 60
    # Where the QA tests store failure logs:
    # qa-tests-backend/src/main/groovy/common/Constants.groovy
    QA_TEST_DEBUG_LOGS = "/tmp/qa-tests-backend-logs"
    QA_SPOCK_RESULTS = "qa-tests-backend/build/spock-reports"
    K8S_LOG_DIR = "/tmp/k8s-service-logs"
    COLLECTOR_METRICS_DIR = "/tmp/collector-metrics"
    DEBUG_OUTPUT = "debug-dump"
    DIAGNOSTIC_OUTPUT = "diagnostic-bundle"
    CENTRAL_DATA_OUTPUT = "central-data"

    def __init__(
        self,
        check_stackrox_logs=False,
        store_qa_test_debug_logs=False,
        store_qa_spock_results=False,
        artifact_destination=None,
    ):
        self._check_stackrox_logs = check_stackrox_logs
        self._store_qa_test_debug_logs = store_qa_test_debug_logs
        self._store_qa_spock_results = store_qa_spock_results
        self.artifact_destination = artifact_destination
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
        self.central_is_responsive = False
        self.data_to_store = []
        if self._store_qa_test_debug_logs:
            self.data_to_store.append(PostClusterTest.QA_TEST_DEBUG_LOGS)
        if self._store_qa_spock_results:
            self.data_to_store.append(PostClusterTest.QA_SPOCK_RESULTS)

    def run(self, test_output_dirs=None):
        self.central_is_responsive = self.wait_for_central_api()
        self.collect_service_logs()
        self.collect_collector_metrics()
        if self.central_is_responsive:
            self.get_central_debug_dump()
            self.get_central_diagnostics()
            self.grab_central_data()
        if self._check_stackrox_logs:
            self.check_stackrox_logs()
        self.store_artifacts(test_output_dirs)
        self.fixup_artifacts_content_type()
        self.make_artifacts_help()
        if self.exitstatus != 0:
            for args in self.failed_commands:
                print(f"Post failure in: {args}")
            raise RuntimeError(f"Post failed: exit {self.exitstatus}")

    def wait_for_central_api(self):
        return self._run_with_best_effort(
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
        self.data_to_store.append(PostClusterTest.K8S_LOG_DIR)

    def collect_collector_metrics(self):
        self._run_with_best_effort(
            [
                "scripts/ci/collect-collector-metrics.sh",
                "stackrox",
                PostClusterTest.COLLECTOR_METRICS_DIR,
            ],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(PostClusterTest.COLLECTOR_METRICS_DIR)

    def get_central_debug_dump(self):
        self._run_with_best_effort(
            [
                "scripts/ci/lib.sh",
                "get_central_debug_dump",
                PostClusterTest.DEBUG_OUTPUT,
            ],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(PostClusterTest.DEBUG_OUTPUT)

    def get_central_diagnostics(self):
        self._run_with_best_effort(
            [
                "scripts/ci/lib.sh",
                "get_central_diagnostics",
                PostClusterTest.DIAGNOSTIC_OUTPUT,
            ],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(PostClusterTest.DIAGNOSTIC_OUTPUT)

    def grab_central_data(self):
        self._run_with_best_effort(
            ["scripts/grab-data-from-central.sh", PostClusterTest.CENTRAL_DATA_OUTPUT],
            timeout=PostClusterTest.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(PostClusterTest.CENTRAL_DATA_OUTPUT)

    def check_stackrox_logs(self):
        self._run_with_best_effort(
            ["tests/e2e/lib.sh", "check_stackrox_logs", PostClusterTest.K8S_LOG_DIR],
            timeout=PostClusterTest.CHECK_TIMEOUT,
        )

    def store_artifacts(self, test_output_dirs):
        for source in test_output_dirs + self.data_to_store:
            args = ["scripts/ci/store-artifacts.sh", "store_artifacts", source]
            if self.artifact_destination:
                args.append(self.artifact_destination)
            self._run_with_best_effort(
                args,
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

    def _run_with_best_effort(self, args: List[str], timeout: int):
        print(f"Running post command: {args}")
        runs_ok = False
        try:
            subprocess.run(
                args,
                check=True,
                timeout=timeout,
            )
            runs_ok = True
        except Exception as err:
            print(f"Exception raised in {args}, {err}")
            self.failed_commands.append(args)
            self.exitstatus = 1
        return runs_ok
