#!/usr/bin/env -S python3 -u

"""
Run integration tests with the compliance operator in a OCP cluster provided
via an openshift/release hive cluster_claim.
"""
import os
from runners import ClusterTestRunner
from clusters import OpenShiftScaleWorkersCluster
from pre_tests import PreSystemTests
from ci_tests import ComplianceE2eTest
from post_tests import PostClusterTest, FinalPost

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["INSTALL_COMPLIANCE_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"

# Scale up the cluster to support postgres
cluster = OpenShiftScaleWorkersCluster(increment=1)

ClusterTestRunner(
    cluster=cluster,
    pre_test=PreSystemTests(),
    test=ComplianceE2eTest(),
    post_test=PostClusterTest(
        check_stackrox_logs=True,
    ),
    final_post=FinalPost(),
).run()
