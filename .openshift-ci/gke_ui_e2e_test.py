#!/usr/bin/env -S python3 -u

"""
Run the UI e2e test in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import UIE2eTest
from post_tests import PostClusterTest, FinalPost

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# Override test env defaults here:
# (for defaults see: tests/e2e/lib.sh export_test_environment())
os.environ["LOAD_BALANCER"] = "none"
os.environ["LOCAL_PORT"] = "8000"
os.environ["OUTPUT_FORMAT"] = "helm"

ClusterTestRunner(
    cluster=GKECluster("ui-e2e-test"),
    pre_test=PreSystemTests(),
    test=UIE2eTest(),
    post_test=PostClusterTest(
        check_stackrox_logs=True,
    ),
    final_post=FinalPost(),
).run()
