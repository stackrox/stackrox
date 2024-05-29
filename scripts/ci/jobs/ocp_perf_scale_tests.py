#!/usr/bin/env -S python3 -u

"""
Run the perf scale test in an OCP cluster
"""
import os
from runners import ClusterTestRunner
from clusters import AutomationFlavorsCluster
from pre_tests import NullPreTest
from ci_tests import NullTest
from post_tests import NullPostTest

os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"

ClusterTestRunner(
    cluster=AutomationFlavorsCluster(),
    pre_test=NullPreTest(),
    test=NullTest(),
    post_test=NullPostTest(),
).run()
