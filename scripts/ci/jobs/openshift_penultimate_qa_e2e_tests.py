#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a openshift 4 cluster provided via a hive cluster_claim.
"""
import os
from runners import ClusterTestRunner
from clusters import OpenShiftScaleWorkersCluster
from pre_tests import PreSystemTests
from ci_tests import UIE2eTest
from post_tests import PostClusterTest, FinalPost

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["OPENSHIFT_CI_CLUSTER_CLAIM"] = "openshift-4"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_POSTGRES_DATASTORE"] = "false"

# Scale up the cluster to support postgres
cluster = OpenShiftScaleWorkersCluster(increment=1)

ClusterTestRunner(
    cluster=cluster,
    pre_test=PreSystemTests(),
    test=UIE2eTest(),
    post_test=PostClusterTest(
        check_stackrox_logs=True,
    ),
    final_post=FinalPost(),
).run()
