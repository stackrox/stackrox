#!/usr/bin/env -S python3 -u

"""
Run operator e2e tests in a openshift 4 cluster provided via a hive cluster_claim.
"""
from runners import ClusterTestRunner
from ci_tests import OperatorE2eTest
from clusters import OpenShiftScaleWorkersCluster
from pre_tests import PreSystemTests
from post_tests import PostClusterTest, FinalPost


ClusterTestRunner(
    cluster=OpenShiftScaleWorkersCluster(),
    pre_test=PreSystemTests(),
    test=OperatorE2eTest(),
    post_test=PostClusterTest(collect_central_artifacts=False),
    final_post=FinalPost(handle_e2e_progress_failures=False),
).run()
