#!/usr/bin/env -S python3 -u

"""
Run tests/e2e in an OCP cluster
"""
import os
from runners import ClusterTestRunner
from clusters import AutomationFlavorsCluster
from pre_tests import PreSystemTests
from ci_tests import NonGroovyE2e
from post_tests import PostClusterTest, FinalPost

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["KUBERNETES_PROVIDER"] = "ocp"
# Include bundles for test images: RHEL (ubi9), Debian (nginx), Alpine (httpd/memcached)
# Delegated scanning mirror tests use nginx/httpd/memcached from quay.io/rhacs-eng/qa
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = (
    "alpine,debian,epss,manual,nvd,osv,rhel-vex,stackrox-rhel-csaf,ubuntu"
)

# delegated scanning support in the secured cluster
os.environ["SENSOR_SCANNER_SUPPORT"] = "true"

# Enable new CRS-based flow for registering secured clusters
os.environ["ROX_DEPLOY_SENSOR_WITH_CRS"] = "true"
os.environ["SENSOR_HELM_MANAGED"] = "true"

ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=PreSystemTests(),
    test=NonGroovyE2e(),
    post_test=PostClusterTest(
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(),
).run()
