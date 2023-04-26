#!/usr/bin/env -S python3 -u

"""
Run operator e2e tests in GKE cluster.
"""
from clusters import GKECluster
from runners import ClusterTestRunner
from ci_tests import OperatorE2eTest
from pre_tests import PreSystemTests
from post_tests import PostClusterTest, FinalPost


ClusterTestRunner(
    cluster=GKECluster("operator-e2e-test"),
    pre_test=PreSystemTests(),
    test=OperatorE2eTest(operator_cluster_type="gke"),
    post_test=PostClusterTest(collect_central_artifacts=False),
    final_post=FinalPost(handle_e2e_progress_failures=False),
).run()
