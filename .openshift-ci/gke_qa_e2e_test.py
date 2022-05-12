#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster
"""
import os
from ci_tests import QaE2eTestPart1, QaE2eTestPart2
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
    sets=[{"test": QaE2eTestPart1()}, {"test": QaE2eTestPart2()}],
).run()
