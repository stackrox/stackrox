#!/usr/bin/env -S python3 -u

"""
Run the upgrade test in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import UpgradeTest
from post_tests import PostClusterTest, FinalPost

# TODO(sbostick): just dev notes — ok to remove
# This entrypoint runs groovy tests via
#   * tests/upgrade/run.sh
#   * tests/upgrade/lib.sh
# Where it uses:
#   * make -C qa-tests-backend smoke-test
#   * make -C qa-tests-backend upgrade-test

# TODO(sbostick): check DEPLOY_DIR
# (hard-coded in tests/upgrade/run.sh to "deploy/k8s")
# might need to override for Openshift secured cluster

os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["SENSOR_HELM_DEPLOY"] = "true"

# don't use postgres
os.environ["ROX_POSTGRES_DATASTORE"] = "false"

ClusterTestRunner(
    cluster=GKECluster("upgrade-test"),
    pre_test=PreSystemTests(),
    test=UpgradeTest(),
    post_test=PostClusterTest(),
    final_post=FinalPost(
        store_qa_test_debug_logs=True,
        store_qa_spock_results=True,
    ),
).run()
