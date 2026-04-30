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
# TODO: Move images to quay.io/rhacs-eng/ before merging to main.
os.environ["VM_IMAGE_RHEL9"] = "quay.io/prygiels/rhel9-dnf-primed:latest"
os.environ["VM_IMAGE_RHEL10"] = "quay.io/prygiels/rhel10-dnf-primed:latest"


class VMScanningPostTest(PostClusterTest):
    """Extends standard post-test with CNV namespace logs and VM artifacts."""

    VM_ARTIFACTS_DIR = "/tmp/vm-scanning-artifacts"

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.k8s_namespaces.extend([
            "openshift-cnv",
        ])

    def run(self, test_outputs=None):
        self.collect_vm_artifacts()
        super().run(test_outputs=test_outputs)

    def collect_vm_artifacts(self):
        """Collect VM/VMI descriptions and events from test namespaces."""
        artifacts_dir = self.VM_ARTIFACTS_DIR
        os.makedirs(artifacts_dir, exist_ok=True)
        for resource in [
            "vm", "vmi", "datavolume",
        ]:
            self.run_with_best_effort(
                [
                    "bash", "-c",
                    f"kubectl get {resource} --all-namespaces -o wide "
                    f"> {artifacts_dir}/{resource}-list.txt 2>&1 || true; "
                    f"kubectl describe {resource} --all-namespaces "
                    f"> {artifacts_dir}/{resource}-describe.txt 2>&1 || true",
                ],
                timeout=self.COLLECT_TIMEOUT,
            )
        self.run_with_best_effort(
            [
                "bash", "-c",
                f"kubectl get events --all-namespaces "
                f"--field-selector involvedObject.kind=VirtualMachineInstance "
                f"> {artifacts_dir}/vmi-events.txt 2>&1 || true",
            ],
            timeout=self.COLLECT_TIMEOUT,
        )
        self.data_to_store.append(artifacts_dir)


ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=PreSystemTests(),
    test=VMScanningE2e(),
    post_test=VMScanningPostTest(
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(),
).run()
