#!/usr/bin/env -S python3 -u

"""
Run nongroovy tests in a GKE arm64 cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import NonGroovyE2e
from post_tests import PostClusterTest, FinalPost

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# delegated scanning support in the secured cluster
os.environ["SENSOR_SCANNER_SUPPORT"] = "true"

# Enable new CRS-based flow for registering secured clusters
os.environ["ROX_DEPLOY_SENSOR_WITH_CRS"] = "true"
os.environ["SENSOR_HELM_MANAGED"] = "true"

# arm64 settings
os.environ["REMOTE_CLUSTER_ARCH"] = "arm64"
os.environ["ARM64_NODESELECTORS"] = "true"
os.environ["IMAGE_PREFETCH_DISABLED"] = "true"

ClusterTestRunner(
    cluster=GKECluster("nongroovy-test", machine_type="t2a-standard-8"),
    pre_test=PreSystemTests(),
    test=NonGroovyE2e(),
    post_test=PostClusterTest(
        check_stackrox_logs=False,
    ),
    final_post=FinalPost(
        store_qa_tests_data=False,
    ),
).run()
