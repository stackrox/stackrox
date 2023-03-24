#!/usr/bin/env -S python3 -u

# temporarily borrows the scale test in order to (a) avoid modifications to
# openshift/release and (b) avoid the powervs allocation step while developing.
# Once dev complete the contents of this file will move to
# powervs_qa_e2e_tests.py.

"""
Run QA e2e tests with sensor & collector deployed to a powervs cluster. Central
will be deployed to a GKE cluster.
"""
import os
from runners import ClusterTestRunner
from clusters import SeparateClusters
from pre_tests import PreSystemTests
from ci_tests import QaE2eTestPart1
from post_tests import PostClusterTest, FinalPost

os.environ["SEPARATE_CLUSTERS_TEST"] = "true"
# For powervs this will be:
# os.environ["CENTRAL_ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"

ClusterTestRunner(
    cluster=SeparateClusters("powervs"),
    # For powervs this will be:
    # cluster=SeparateClusters("powervs", sensor_cluster_kubeconfig=os.environ["KUBECONFIG"]),
    pre_test=PreSystemTests(),
    test=QaE2eTestPart1(),
    post_test=PostClusterTest(
        check_stackrox_logs=True,
    ),
    final_post=FinalPost(
        store_qa_test_debug_logs=True,
        store_qa_spock_results=True,
    ),
).run()
