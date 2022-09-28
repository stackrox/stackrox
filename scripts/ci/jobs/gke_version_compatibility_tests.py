#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import os

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

versions=["3.71.0", "3.70.0", "3.69.0"]

gkecluster=GKECluster("qa-e2e-test")

for version in versions:
    os.environ["SENSOR_IMAGE_TAG"] = version
    make_compatibility_test_runner(cluster=gkecluster).run()

print("stub for version compatibility tests")
