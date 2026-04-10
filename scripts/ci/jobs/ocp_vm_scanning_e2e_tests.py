#!/usr/bin/env -S python3 -u

"""
Run VM scanning E2E tests on an OpenShift cluster with CNV / KubeVirt and VSOCK enabled.

Cluster contract: a KubeVirt custom resource must exist in a standard CNV namespace with the
VSOCK feature gate set, and virt-handler pods must mount host paths that expose vsock plumbing.
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

ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=PreSystemTests(),
    test=VMScanningE2e(),
    post_test=PostClusterTest(
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(),
).run()
