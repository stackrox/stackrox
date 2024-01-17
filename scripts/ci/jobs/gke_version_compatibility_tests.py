#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import logging
import os
import subprocess
import sys

from collections import namedtuple
from pathlib import Path

from pre_tests import PreSystemTests
from ci_tests import QaE2eTestCompatibility
from post_tests import PostClusterTest, FinalPost
from runners import ClusterTestSetsRunner
from clusters import GKECluster
from get_latest_helm_chart_versions import (
    get_latest_helm_chart_versions,
    get_latest_helm_chart_version_for_specific_release,
)

Release = namedtuple("Release", ["major", "minor"])

# start logging
logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"

central_chart_versions = get_latest_helm_chart_versions(
    "stackrox-central-services", 2)
sensor_chart_versions = get_latest_helm_chart_versions(
    "stackrox-secured-cluster-services", 3
)
makefile_path = Path(__file__).parent.parent.parent.parent
latest_tag = subprocess.check_output(
    ["make", "tag", "-C", makefile_path, "--quiet", "--no-print-director"],
    shell=False,
    encoding="utf-8",
).strip()

if len(central_chart_versions) == 0:
    raise RuntimeError("Could not find central chart versions.")
if len(sensor_chart_versions) == 0:
    raise RuntimeError("Could not find sensor chart versions.")

ChartVersions = namedtuple(
    "Chart_versions", ["central_version", "sensor_version"])

# Latest central vs sensor versions in sensor_chart_versions
test_tuples = [
    ChartVersions(central_version=latest_tag,
                  sensor_version=sensor_chart_version)
    for sensor_chart_version in sensor_chart_versions
]
# Latest sensor vs central versions in central_chart_versions
test_tuples.extend(
    [
        ChartVersions(central_version=central_chart_version,
                      sensor_version=latest_tag)
        for central_chart_version in central_chart_versions
    ]
)

# Support exception for latest central and sensor 3.74 as per
# https://issues.redhat.com/browse/ROX-18223
support_exceptions = [
    ChartVersions(
        central_version=latest_tag,
        sensor_version=get_latest_helm_chart_version_for_specific_release(
            "stackrox-secured-cluster-services", Release(major=3, minor=74)
        ),
    )
]

test_tuples.extend(
    support_exception
    for support_exception in support_exceptions
    if support_exception not in test_tuples
)

sets = []
for test_tuple in test_tuples:
    os.environ["ROX_TELEMETRY_STORAGE_KEY_V1"] = 'DISABLED'
    test_versions = f'{test_tuple.central_version}--{test_tuple.sensor_version}'
    sets.append(
        {
            "name": f'version compatibility tests: {test_versions}',
            "test": QaE2eTestCompatibility(test_tuple.central_version, test_tuple.sensor_version),
            "post_test": PostClusterTest(
                    check_stackrox_logs=True,
                    artifact_destination_prefix=test_versions,
            ),
        },
    )
sets[0]["pre_test"] = PreSystemTests()

ClusterTestSetsRunner(
    cluster=GKECluster("compat-test",
                       machine_type="e2-standard-8", num_nodes=2),
    sets=sets,
    final_post=FinalPost(
        store_qa_tests_data=True,
    ),
).run()
