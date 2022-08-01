#!/usr/bin/env -S python3 -u

"""
Run operator e2e tests in a openshift 4 cluster provided via a hive cluster_claim.
"""
from runners import ClusterTestRunner
from ci_tests import OperatorE2eTest
from pre_tests import PreSystemTests
from post_tests import PostClusterTest, FinalPost


ClusterTestRunner(
    pre_test=PreSystemTests(),
    test=OperatorE2eTest(),
    post_test=PostClusterTest(),
    final_post=FinalPost(
        store_qa_test_debug_logs=True,
        store_qa_spock_results=True,
    ),
).run()
