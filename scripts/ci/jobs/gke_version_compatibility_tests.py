#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import logging
import os
import subprocess
import sys
from clusters import GKECluster
from collections import namedtuple
from compatibility_test import make_compatibility_test_runner
from get_latest_helm_chart_versions import get_latest_helm_chart_versions

# start logging
logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"

central_chart_versions = get_latest_helm_chart_versions("stackrox-central-services", 2)
sensor_chart_versions = get_latest_helm_chart_versions("stackrox-secured-cluster-services", 3)
makefile_path = os.path.abspath(os.path.join(os.path.dirname( __file__ ), '../../..'))
latest_tag = subprocess.check_output(["make", "tag", "-C", makefile_path, "--quiet", "--no-print-director"], shell=False, encoding='utf-8').strip()

if len(central_chart_versions) == 0:
    raise RuntimeError("Could not find central chart versions.")
if len(sensor_chart_versions) == 0:
    raise RuntimeError("Could not find sensor chart versions.")

Chart_versions = namedtuple("Chart_versions", ["central_version", "sensor_version"])

# Latest central vs sensor versions in sensor_chart_versions (latest 4 releases)
test_tuples = [Chart_versions(central_version=latest_tag, sensor_version=sensor_chart_version) for sensor_chart_version in sensor_chart_versions]
# Latest sensor vs central versions in central_chart_versions (latest 2 releases)
test_tuples.extend([Chart_versions(central_version=central_chart_version, sensor_version=latest_tag) for central_chart_version in central_chart_versions])

gkecluster = GKECluster("compat-test")

failing_tuples = []
for tuple in test_tuples:
    central_version = tuple[0]
    sensor_version = tuple[1]
    os.environ["CENTRAL_CHART_VERSION_OVERRIDE"] = central_version
    os.environ["SENSOR_CHART_VERSION_OVERRIDE"] = sensor_version
    try:
        make_compatibility_test_runner(cluster=gkecluster).run()
    except Exception:
        print(f"Exception \"{Exception}\" raised in compatibility test for central version {central_version} and sensor version {sensor_version}",
            file=sys.stderr)
        failing_tuples.append(tuple)

if len(failing_tuples) > 0:
    failing_string = ', '.join([("(Central v" + str(failing_tuple["central_version"]) + ", Sensor v" + str(failing_tuple["sensor_version"]) + ")") for failing_tuple in failing_tuples])
    raise RuntimeError("Compatibility tests failed for versions " + failing_string + ".")
