#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster with external PG 17
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import BYODBTest
from post_tests import PostClusterTest, FinalPost

# set test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["POSTGRES_VERSION"] = "17"
os.environ["BYODB_TEST"] = "true"

os.environ["ROX_ACTIVE_VULN_MGMT"] = "true"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

ClusterTestRunner(
    cluster=GKECluster("byodb-test", machine_type="e2-standard-8"),
    pre_test=PreSystemTests(),
    test=BYODBTest(),
    post_test=PostClusterTest(
        collect_central_artifacts=False
    ),
    final_post=FinalPost(
        store_qa_tests_data=True,
    ),
).run()
