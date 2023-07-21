#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import logging
import os
import sys

from clusters import GKECluster
from compatibility_test import make_compatibility_test_runner
from get_latest_helm_chart_versions import get_latest_helm_chart_versions

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"

logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

chart_versions = get_latest_helm_chart_versions("stackrox-secured-cluster-services")

gkecluster = GKECluster("compat-test", num_nodes=2, machine_type="e2-standard-8")

failing_sensor_versions = []
for version in chart_versions:
    os.environ["SENSOR_CHART_VERSION"] = version
    try:
        make_compatibility_test_runner(cluster=gkecluster).run()
    except Exception:
        print(f"Exception \"{Exception}\" raised in compatibility test for sensor chart version {version}",
              file=sys.stderr)
        failing_sensor_versions += version

if len(failing_sensor_versions) > 0:
    raise RuntimeError(f"Compatibility tests failed for Sensor versions " + ', '.join(failing_sensor_versions))
