#!/usr/bin/env -S python3 -u

"""
Run tests/e2e in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import NonGroovyE2e
from post_tests import PostClusterTest, FinalPost

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["KUBERNETES_PROVIDER"] = "gke"
os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "faster"
# On GKE only ubi9 (RHEL) images are scanned with vuln assertions (delegated_scanning_test.go).
# Matches ocp_vm_scanning_e2e_tests.py allowlist — drops alpine, debian, osv, ubuntu.
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = "rhel-vex,stackrox-rhel-csaf,manual,epss,nvd"

# delegated scanning support in the secured cluster
os.environ["SENSOR_SCANNER_SUPPORT"] = "true"

# Enable new CRS-based flow for registering secured clusters
os.environ["ROX_DEPLOY_SENSOR_WITH_CRS"] = "true"
os.environ["SENSOR_HELM_MANAGED"] = "true"

ClusterTestRunner(
    cluster=GKECluster("nongroovy-test"),
    pre_test=PreSystemTests(),
    test=NonGroovyE2e(),
    post_test=PostClusterTest(
        check_stackrox_logs=True,
    ),
    final_post=FinalPost(
        store_qa_tests_data=False,
    ),
).run()
