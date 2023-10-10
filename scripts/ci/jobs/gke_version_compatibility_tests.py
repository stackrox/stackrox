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
from get_latest_helm_chart_versions import get_latest_helm_chart_version_for_specific_release
from pathlib import Path

Release = namedtuple("Release", ["major", "minor"])

# start logging
logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"

central_chart_versions = get_latest_helm_chart_versions("stackrox-central-services", 2)
sensor_chart_versions = get_latest_helm_chart_versions("stackrox-secured-cluster-services", 3)
makefile_path = Path(__file__).parent.parent.parent.parent
latest_tag = subprocess.check_output(["make", "tag", "-C", makefile_path, "--quiet", "--no-print-director"], shell=False, encoding='utf-8').strip()

if len(central_chart_versions) == 0:
    raise RuntimeError("Could not find central chart versions.")
if len(sensor_chart_versions) == 0:
    raise RuntimeError("Could not find sensor chart versions.")

Chart_versions = namedtuple("Chart_versions", ["central_version", "sensor_version"])

# Latest central vs sensor versions in sensor_chart_versions
test_tuples = [Chart_versions(central_version=latest_tag, sensor_version=sensor_chart_version) for sensor_chart_version in sensor_chart_versions]
# Latest sensor vs central versions in central_chart_versions
test_tuples.extend([Chart_versions(central_version=central_chart_version, sensor_version=latest_tag) for central_chart_version in central_chart_versions])

# Support exception for latest central and sensor 3.74 as per https://issues.redhat.com/browse/ROX-18223
support_exceptions = [Chart_versions(central_version=latest_tag, sensor_version=get_latest_helm_chart_version_for_specific_release("stackrox-secured-cluster-services", Release(major=3, minor=74)))]

test_tuples.extend(support_exception for support_exception in support_exceptions if support_exception not in test_tuples)

gkecluster = GKECluster("compat-test", machine_type="e2-standard-8", num_nodes=2)

failing_tuples = []
for tuple in test_tuples:
    os.environ["CENTRAL_CHART_VERSION_OVERRIDE"] = tuple.central_version
    os.environ["SENSOR_CHART_VERSION_OVERRIDE"] = tuple.sensor_version
    try:
        make_compatibility_test_runner(cluster=gkecluster).run()
    except Exception as e:
        print(f"Exception \"{str(e)}\" raised in compatibility test for central version {tuple.central_version} and sensor version {tuple.sensor_version}",
            file=sys.stderr)
        failing_tuples.append(tuple)

if len(failing_tuples) > 0:
    failing_string = ', '.join([("(Central v" + str(failing_tuple.central_version) + ", Sensor v" + str(failing_tuple.sensor_version) + ")") for failing_tuple in failing_tuples])
    raise RuntimeError("Compatibility tests failed for versions " + failing_string + ".")
