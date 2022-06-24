#!/usr/bin/env python3

"""
Common steps to run when e2e tests are complete. All post steps are run in spite
of prior failures. This models existing CI behavior from Circle CI.
"""

import os
import subprocess
from typing import List


class PostTestsConstants:

    API_TIMEOUT = 5 * 60
    COLLECT_TIMEOUT = 5 * 60
    CHECK_TIMEOUT = 5 * 60
    STORE_TIMEOUT = 5 * 60
    FIXUP_TIMEOUT = 5 * 60
    # Where the QA tests store failure logs:
    # qa-tests-backend/src/main/groovy/common/Constants.groovy
    QA_TEST_DEBUG_LOGS = "/tmp/qa-tests-backend-logs"
    QA_SPOCK_RESULTS = "qa-tests-backend/build/spock-reports"
    K8S_LOG_DIR = "/tmp/k8s-service-logs"
    COLLECTOR_METRICS_DIR = "/tmp/collector-metrics"
    DEBUG_OUTPUT = "debug-dump"
    DIAGNOSTIC_OUTPUT = "diagnostic-bundle"
    CENTRAL_DATA_OUTPUT = "central-data"
    STACKROX_LOG_DIR = "/tmp/stackrox-logs"


class NullPostTest:
    def run(self, test_output_dirs=None):
        pass


class RunWithBestEffortMixin:
    def __init__(
        self,
    ):
        self.exitstatus = 0
        self.failed_commands: List[List[str]] = []

    def run_with_best_effort(self, args: List[str], timeout: int):
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

    def handle_run_failure(self):
        if self.exitstatus != 0:
            for args in self.failed_commands:
                print(f"Post failure in: {args}")
            raise RuntimeError(f"Post failed: exit {self.exitstatus}")


class StoreArtifacts(RunWithBestEffortMixin):
    """For tests that only need to store artifacts"""

    def __init__(
        self,
        artifact_destination_prefix=None,
    ):
        super().__init__()
        self.artifact_destination_prefix = artifact_destination_prefix
        self.data_to_store = []

    def run(self, test_output_dirs=None):
        self.store_artifacts(test_output_dirs)
        self.handle_run_failure()

    def store_artifacts(self, test_output_dirs=None):
        if test_output_dirs is not None:
            self.data_to_store = test_output_dirs + self.data_to_store
        for source in self.data_to_store:
            args = ["scripts/ci/store-artifacts.sh", "store_artifacts", source]
            if self.artifact_destination_prefix:
                args.append(
                    os.path.join(
                        self.artifact_destination_prefix, os.path.basename(source)
                    )
                )
            self.run_with_best_effort(
                args,
                timeout=PostTestsConstants.STORE_TIMEOUT,
            )


# pylint: disable=too-many-instance-attributes
class PostClusterTest(StoreArtifacts):
    """The standard cluster test suite of debug gathering and analysis"""

    def __init__(
        self,
        check_stackrox_logs=False,
        artifact_destination_prefix=None,
    ):
        super().__init__(artifact_destination_prefix=artifact_destination_prefix)
        self._check_stackrox_logs = check_stackrox_logs
        self.k8s_namespaces = ["stackrox", "stackrox-operator", "proxies", "squid"]
        self.openshift_namespaces = [
            "openshift-dns",
            "openshift-apiserver",
            "openshift-authentication",
            "openshift-etcd",
            "openshift-controller-manager",
        ]
        self.central_is_responsive = False

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
        self.handle_run_failure()

    def wait_for_central_api(self):
        return self.run_with_best_effort(
            ["tests/e2e/lib.sh", "wait_for_api"],
            timeout=PostTestsConstants.API_TIMEOUT,
        )

    def collect_service_logs(self):
        for namespace in self.k8s_namespaces + self.openshift_namespaces:
            self.run_with_best_effort(
                [
                    "scripts/ci/collect-service-logs.sh",
                    namespace,
                    PostTestsConstants.K8S_LOG_DIR,
                ],
                timeout=PostTestsConstants.COLLECT_TIMEOUT,
            )
        self.run_with_best_effort(
            [
                "scripts/ci/collect-infrastructure-logs.sh",
                PostTestsConstants.K8S_LOG_DIR,
            ],
            timeout=PostTestsConstants.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(PostTestsConstants.K8S_LOG_DIR)

    def collect_collector_metrics(self):
        self.run_with_best_effort(
            [
                "scripts/ci/collect-collector-metrics.sh",
                "stackrox",
                PostTestsConstants.COLLECTOR_METRICS_DIR,
            ],
            timeout=PostTestsConstants.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(PostTestsConstants.COLLECTOR_METRICS_DIR)

    def get_central_debug_dump(self):
        self.run_with_best_effort(
            [
                "scripts/ci/lib.sh",
                "get_central_debug_dump",
                PostTestsConstants.DEBUG_OUTPUT,
            ],
            timeout=PostTestsConstants.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(PostTestsConstants.DEBUG_OUTPUT)

    def get_central_diagnostics(self):
        self.run_with_best_effort(
            [
                "scripts/ci/lib.sh",
                "get_central_diagnostics",
                PostTestsConstants.DIAGNOSTIC_OUTPUT,
            ],
            timeout=PostTestsConstants.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(PostTestsConstants.DIAGNOSTIC_OUTPUT)

    def grab_central_data(self):
        self.run_with_best_effort(
            [
                "scripts/grab-data-from-central.sh",
                PostTestsConstants.CENTRAL_DATA_OUTPUT,
            ],
            timeout=PostTestsConstants.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(PostTestsConstants.CENTRAL_DATA_OUTPUT)

    def check_stackrox_logs(self):
        self.run_with_best_effort(
            ["tests/e2e/lib.sh", "check_stackrox_logs", PostTestsConstants.K8S_LOG_DIR],
            timeout=PostTestsConstants.CHECK_TIMEOUT,
        )


class CheckStackroxLogs(StoreArtifacts):
    """When only stackrox logs and checks are required"""

    def __init__(
        self,
        check_for_stackrox_restarts=False,
        check_for_errors_in_stackrox_logs=False,
        artifact_destination_prefix=None,
    ):
        super().__init__(artifact_destination_prefix=artifact_destination_prefix)
        self._check_for_stackrox_restarts = check_for_stackrox_restarts
        self._check_for_errors_in_stackrox_logs = check_for_errors_in_stackrox_logs
        self.central_is_responsive = False

    def run(self, test_output_dirs=None):
        self.central_is_responsive = self.wait_for_central_api()
        if self.central_is_responsive:
            self.collect_stackrox_logs()
            if self._check_for_stackrox_restarts:
                self.check_for_stackrox_restarts()
            if self._check_for_errors_in_stackrox_logs:
                self.check_for_errors_in_stackrox_logs()
        self.store_artifacts(test_output_dirs)
        self.handle_run_failure()

    def wait_for_central_api(self):
        return self.run_with_best_effort(
            ["tests/e2e/lib.sh", "wait_for_api"],
            timeout=PostTestsConstants.API_TIMEOUT,
        )

    def collect_stackrox_logs(self):
        self.run_with_best_effort(
            [
                "scripts/ci/collect-service-logs.sh",
                "stackrox",
                PostTestsConstants.STACKROX_LOG_DIR,
            ],
            timeout=PostTestsConstants.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(PostTestsConstants.STACKROX_LOG_DIR)

    def check_for_stackrox_restarts(self):
        self.run_with_best_effort(
            [
                "tests/e2e/lib.sh",
                "check_for_stackrox_restarts",
                PostTestsConstants.STACKROX_LOG_DIR,
            ],
            timeout=PostTestsConstants.CHECK_TIMEOUT,
        )

    def check_for_errors_in_stackrox_logs(self):
        self.run_with_best_effort(
            [
                "tests/e2e/lib.sh",
                "check_for_errors_in_stackrox_logs",
                PostTestsConstants.STACKROX_LOG_DIR,
            ],
            timeout=PostTestsConstants.CHECK_TIMEOUT,
        )


class FinalPost(StoreArtifacts):
    """Collect logs that accumulate over multiple tests and other final steps"""

    def __init__(
        self,
        store_qa_test_debug_logs=False,
        store_qa_spock_results=False,
        artifact_destination_prefix="final",
    ):
        super().__init__(artifact_destination_prefix=artifact_destination_prefix)
        self._store_qa_test_debug_logs = store_qa_test_debug_logs
        self._store_qa_spock_results = store_qa_spock_results
        if self._store_qa_test_debug_logs:
            self.data_to_store.append(PostTestsConstants.QA_TEST_DEBUG_LOGS)
        if self._store_qa_spock_results:
            self.data_to_store.append(PostTestsConstants.QA_SPOCK_RESULTS)

    def run(self, test_output_dirs=None):
        self.store_artifacts()
        self.fixup_artifacts_content_type()
        self.make_artifacts_help()
        self.handle_run_failure()

    def fixup_artifacts_content_type(self):
        self.run_with_best_effort(
            ["scripts/ci/store-artifacts.sh", "fixup_artifacts_content_type"],
            timeout=PostTestsConstants.FIXUP_TIMEOUT,
        )

    def make_artifacts_help(self):
        self.run_with_best_effort(
            ["scripts/ci/store-artifacts.sh", "make_artifacts_help"],
            timeout=PostTestsConstants.FIXUP_TIMEOUT,
        )
