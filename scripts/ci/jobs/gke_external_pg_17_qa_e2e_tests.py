#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster with external PG 17
"""
# import os
# from base_qa_e2e_test import make_qa_e2e_test_runner
# from clusters import GKECluster
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import BYODBTest
from post_tests import PostClusterTest, FinalPost

# set test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["GCP_IMAGE_TYPE"] = "cos_containerd"

os.environ["ROX_ACTIVE_VULN_MGMT"] = "true"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# make_qa_e2e_test_runner(cluster=GKECluster("qa-external-pg-17-e2e-test")).run()
ClusterTestRunner(
    cluster=GKECluster("byodb-test", machine_type="e2-standard-8"),
    pre_test=PreSystemTests(),
    test=BYODBTest(),
    post_test=PostClusterTest(),
    final_post=FinalPost(
        store_qa_tests_data=True,
    ),
).run()
