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
os.environ["INSTALL_CNV_OPERATOR"] = "true"
os.environ["ROX_VIRTUAL_MACHINES"] = "true"
os.environ["ROX_SCANNER_V4"] = "true"
os.environ["VM_IMAGES"] = ",".join([
    "quay.io/rhacs-eng/vm-images/rhel9-dnf-primed-latest",
    "quay.io/rhacs-eng/vm-images/rhel10-dnf-primed-latest",
])


class VMScanningPostTest(PostClusterTest):
    """Extends standard post-test with CNV namespace logs."""

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.k8s_namespaces.extend([
            "openshift-cnv",
        ])


ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=PreSystemTests(),
    test=VMScanningE2e(),
    post_test=VMScanningPostTest(
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(),
).run()
