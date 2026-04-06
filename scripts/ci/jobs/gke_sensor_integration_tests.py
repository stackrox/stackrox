#!/usr/bin/env -S python3 -u

"""
Run sensor-integration tests in a GKE cluster
"""
import os
import scanner_v4_defaults
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import SensorIntegration

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# Scanner V4
os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "faster"
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = scanner_v4_defaults.VULN_BUNDLE_ALLOWLIST

ClusterTestRunner(
    pre_test=PreSystemTests(run_poll_for_system_test_images=False),
    cluster=GKECluster("sensor-integration-test"),
    test=SensorIntegration(),
).run()
