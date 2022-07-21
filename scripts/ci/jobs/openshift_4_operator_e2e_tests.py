#!/usr/bin/env -S python3 -u

"""
Run operator e2e tests in a openshift 4 cluster provided via a hive cluster_claim.
"""
from runners import ClusterTestRunner
from ci_tests import OperatorE2eTest
from pre_tests import PreSystemTests


ClusterTestRunner(
    pre_test=PreSystemTests(),
    test=OperatorE2eTest()
).run()
