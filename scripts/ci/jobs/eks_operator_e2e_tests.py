#!/usr/bin/env -S python3 -u

"""
Run tests/e2e in an EKS cluster provided via automation-flavors/eks.
"""
import os
from runners import ClusterTestRunner
from clusters import AutomationFlavorsCluster
from pre_tests import PreSystemTests
from ci_tests import OperatorE2eTest
from post_tests import PostClusterTest, FinalPost

os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["KUBERNETES_PROVIDER"] = "eks"
# On k8s only ubi9 (RHEL) images are scanned with vuln assertions (delegated_scanning_test.go).
# Matches gke_nongroovy_e2e_tests.py allowlist — drops alpine, debian, osv, ubuntu.
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = "rhel-vex,stackrox-rhel-csaf,manual,epss,nvd"

# delegated scanning support in the secured cluster
os.environ["SENSOR_SCANNER_SUPPORT"] = "true"

# Enable new CRS-based flow for registering secured clusters
os.environ["ROX_DEPLOY_SENSOR_WITH_CRS"] = "true"
os.environ["SENSOR_HELM_MANAGED"] = "true"

ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=PreSystemTests(),
    test=OperatorE2eTest(operator_cluster_type="eks"),
    post_test=PostClusterTest(collect_central_artifacts=False),
    final_post=FinalPost(),
).run()
