#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import logging
import os
import sys
import subprocess
from clusters import GKECluster
from compatibility_test import make_compatibility_test_runner
from get_latest_helm_chart_versions import get_latest_helm_chart_versions


# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"

central_chart_versions = get_latest_helm_chart_versions("stackrox-central-services", 2)
sensor_chart_versions = get_latest_helm_chart_versions("stackrox-secured-cluster-services", 3)
latest_tag = subprocess.check_output("make tag", shell=True).strip().decode("utf-8")

if len(central_chart_versions) == 0:
    raise RuntimeError("Could not find central chart versions.")
# Latest central vs last 4 sensor versions
test_tuples = [[latest_tag, sensor_chart_versions[i]] for i in range(0, len(sensor_chart_versions))]
# Latest sensor vs 1 version older central
if len(central_chart_versions) > 1:
    test_tuples.append([central_chart_versions[1], latest_tag])

logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

chart_versions = get_latest_helm_chart_versions("stackrox-secured-cluster-services")

gkecluster = GKECluster("compat-test")

failing_tuples = []
for tuple in test_tuples:
    central_version = tuple[0]
    sensor_version = tuple[1]
    os.environ["CENTRAL_CHART_VERSION"] = central_version
    os.environ["SENSOR_CHART_VERSION"] = sensor_version
    try:
        make_compatibility_test_runner(cluster=gkecluster).run()
    except Exception:
        print(f"Exception \"{Exception}\" raised in compatibility test for central version {central_version} and sensor version {sensor_version}",
            file=sys.stderr)
        failing_tuples.append(tuple)

if len(failing_tuples) > 0:
    failing_string = ', '.join([("(Central v" + str(failing_tuples[i][0]) + ", Sensor v" + str(failing_tuples[i][1]) + ")") for i in range(0, len(failing_tuples))])
    raise RuntimeError("Compatibility tests failed for versions " + failing_string + ".")
