#!/usr/bin/env -S python3 -u

"""
Run tests/e2e in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from pre_tests import PreSystemTests
from ci_tests import SensorIntegration
from post_tests import PostClusterTest, FinalPost

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_ACTIVE_VULN_MGMT"] = "true"

ClusterTestRunner(
    pre_test=PreSystemTests(),
    test=SensorIntegration(),
    post_test=PostClusterTest(collect_central_artifacts=False),
    final_post=FinalPost(
        store_qa_test_debug_logs=False,
        store_qa_spock_results=False,
    ),
).run()
