#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import os
from get_latest_release_versions import get_latest_release_versions
from compatibility_test import make_compatibility_test_runner
from clusters import GKECluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# don't use postgres
os.environ["ROX_POSTGRES_DATASTORE"] = "false"

versions=get_latest_release_versions(4)

gkecluster=GKECluster("qa-e2e-test")

failing_sensor_versions = []
for version in versions:
    os.environ["SENSOR_IMAGE_TAG"] = version
    try:
        make_compatibility_test_runner(cluster=gkecluster).run()
    except Exception:
        print(f"Exception \"{Exception}\" raised in compatibility test for sensor version {version}")
        failing_sensor_versions += version

if len(failing_sensor_versions) > 0:
    raise SensorVersionsFailure(f"Compatibility tests failed for Sensor versions {failing_sensor_versions}.")

class SensorVersionsFailure(Exception):
    pass
