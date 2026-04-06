#!/usr/bin/env -S python3 -u

"""
Run the UI e2e test in a GKE cluster
"""
import os
import scanner_v4_defaults
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import UIE2eTest
from post_tests import PostClusterTest, FinalPost

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# Override test env defaults here:
# (for defaults see: tests/e2e/lib.sh export_test_environment())
os.environ["OUTPUT_FORMAT"] = "helm"
os.environ["SENSOR_SCANNER_SUPPORT"] = "true"

# Scanner V4
os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "stackrox-gke-ssd"
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = scanner_v4_defaults.VULN_BUNDLE_ALLOWLIST

ClusterTestRunner(
    cluster=GKECluster("ui-e2e-test"),
    pre_test=PreSystemTests(),
    test=UIE2eTest(),
    post_test=PostClusterTest(
        check_stackrox_logs=True,
    ),
    final_post=FinalPost(),
).run()
