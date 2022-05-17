#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster
"""
import os
from pre_tests import PreSystemTests
from ci_tests import QaE2eTestPart1, QaE2eTestPart2, QaE2eDBBackupRestoreTest
from post_tests import PostClusterTest, StoreArtifacts
from clusters import GKECluster
from runners import ClusterTestSetsRunner

os.environ["ADMISSION_CONTROLLER_UPDATES"] = "true"
os.environ["ADMISSION_CONTROLLER"] = "true"
os.environ["COLLECTION_METHOD"] = "ebpf"
os.environ["GCP_IMAGE_TYPE"] = "COS"
os.environ["LOAD_BALANCER"] = "lb"
os.environ["LOCAL_PORT"] = "443"
os.environ["MONITORING_SUPPORT"] = "false"
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["ROX_BASELINE_GENERATION_DURATION"] = "1m"
os.environ["ROX_NETWORK_BASELINE_OBSERVATION_PERIOD"] = "2m"
os.environ["ROX_NEW_POLICY_CATEGORIES"] = "true"
os.environ["SCANNER_SUPPORT"] = "true"

ClusterTestSetsRunner(
    cluster=GKECluster("qa-e2e-test"),
    sets=[
        {
            "pre_test": PreSystemTests(),
            "test": QaE2eTestPart1(),
            "post_test": PostClusterTest(
                check_stackrox_logs=True, artifact_destination="part-1"
            ),
        },
        {
            "test": QaE2eTestPart2(),
            "post_test": PostClusterTest(
                check_stackrox_logs=True,
                store_qa_test_debug_logs=True,
                store_qa_spock_results=True,
                artifact_destination="part-2",
            ),
        },
        {
            "test": QaE2eDBBackupRestoreTest(),
            "post_test": StoreArtifacts(),
        },
    ],
).run()
