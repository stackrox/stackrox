#!/usr/bin/env -S python3 -u

"""
Run VM scanning E2E tests on an OpenShift cluster with CNV / KubeVirt and VSOCK enabled.

The CNV operator is installed automatically when INSTALL_CNV_OPERATOR=true (set below).
If already present on the cluster, the existing installation is reused.
VSOCK prerequisites are verified at Go test startup (mustVerifyClusterVSOCKReady) and the
suite fails fast with actionable diagnostics if they are not met.
"""
import os

from runners import ClusterTestRunner
from clusters import AutomationFlavorsCluster
from pre_tests import PreSystemTests
from ci_tests import VMScanningE2e
from post_tests import PostClusterTest, FinalPost

os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["SENSOR_SCANNER_SUPPORT"] = "true"
os.environ["ROX_DEPLOY_SENSOR_WITH_CRS"] = "true"
os.environ["SENSOR_HELM_MANAGED"] = "true"
os.environ["VM_SCAN_REQUIRE_ACTIVATION"] = "true"
os.environ["INSTALL_CNV_OPERATOR"] = "true"

ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=PreSystemTests(),
    test=VMScanningE2e(),
    post_test=PostClusterTest(
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(),
).run()
