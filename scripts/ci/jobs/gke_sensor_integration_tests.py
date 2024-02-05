#!/usr/bin/env -S python3 -u

"""
Run sensor-integration tests in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import SensorIntegration
from post_tests import PostClusterTest, FinalPost

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

ClusterTestRunner(
    pre_test=PreSystemTests(run_poll_for_system_test_images=False),
    cluster=GKECluster("sensor-integration-test"),
    test=SensorIntegration(),
    post_test=PostClusterTest(
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(
        store_qa_tests_data=False,
    ),
).run()
