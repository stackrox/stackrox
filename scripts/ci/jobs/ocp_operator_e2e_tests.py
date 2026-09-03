#!/usr/bin/env -S python3 -u

"""
Run operator e2e tests in an OCP cluster.
"""
import os
from runners import ClusterTestRunner
from clusters import AutomationFlavorsCluster
from ci_tests import OperatorE2eTest
from pre_tests import PreSystemTests
from post_tests import PostClusterTest, FinalPost

os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["KUBERNETES_PROVIDER"] = "ocp"
# Optimize Scanner V4 startup time by loading only RHEL vulnerability bundles
# Operator tests only deploy RHEL/UBI9 images (StackRox platform components)
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = "rhel-vex,stackrox-rhel-csaf,manual,epss,nvd"

ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=PreSystemTests(),
    test=OperatorE2eTest(),
    post_test=PostClusterTest(collect_central_artifacts=False),
    final_post=FinalPost(),
).run()
