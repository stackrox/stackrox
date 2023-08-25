#!/usr/bin/env -S python3 -u

"""
Run the scale test in a GKE cluster
"""
import os
from runners import ClusterTestRunner
from clusters import GKECluster
from pre_tests import PreSystemTests
from ci_tests import ScaleTest
from post_tests import PostClusterTest, FinalPost

os.environ["COMPARISON_METRICS"] = "scale-test/gke"
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["OUTPUT_FORMAT"] = "helm"
os.environ["STORAGE"] = "pvc"
os.environ["STORAGE_CLASS"] = "faster"
os.environ["STORAGE_SIZE"] = "100"
os.environ["STORE_METRICS"] = os.environ["COMPARISON_METRICS"]
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_BASELINE_GENERATION_DURATION"] = "5m"

ClusterTestRunner(
    cluster=GKECluster("scale-test", machine_type="e2-standard-8"),
    pre_test=PreSystemTests(),
    test=ScaleTest(),
    post_test=PostClusterTest(
        check_stackrox_logs=True,
    ),
    final_post=FinalPost(),
).run()
