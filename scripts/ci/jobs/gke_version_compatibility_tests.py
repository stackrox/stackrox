#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import json
import logging
import os
import subprocess
import sys
from collections import namedtuple
from pathlib import Path
import requests

from pre_tests import (
    PreSystemTests,
    CollectionMethodOverridePreTest
)
from ci_tests import QaE2eTestCompatibility
from post_tests import PostClusterTest, FinalPost
from runners import ClusterTestSetsRunner
from clusters import GKECluster
from get_latest_helm_chart_versions import (
    get_latest_helm_chart_version_for_specific_release,
)

Release = namedtuple("Release", ["major", "minor"])

# start logging
logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"


def get_supported_versions():
    supported_central = []
    supported_sensor = []
    response = requests.get("https://access.redhat.com/product-life-cycles/api/v1/products?name="
                            "Red%20Hat%20Advanced%20Cluster%20Security%20for%20Kubernetes")
    data = json.loads(response.text)
    versions = data["data"][0]["versions"]
    for version in versions:
        if version["type"] != "End of life":
            major = version["name"].split('.')[0]
            minor = version["name"].split('.')[1]
            supported_central.append(get_latest_helm_chart_version_for_specific_release(
                "stackrox-central-services", Release(major=major, minor=minor)
            ))
            supported_sensor.append(get_latest_helm_chart_version_for_specific_release(
                "stackrox-secured-cluster-services", Release(major=major, minor=minor)
            ))
    return supported_central, supported_sensor


central_chart_versions, sensor_chart_versions = get_supported_versions()

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

    # expected version string is like 74.x.x for ACS 3.74 versions
    is_3_74_sensor = test_tuple.sensor_version.startswith('74')

    sets.append(
        {
            "name": f'version compatibility tests: {test_versions}',
            "test": QaE2eTestCompatibility(test_tuple.central_version, test_tuple.sensor_version),
            "post_test": PostClusterTest(
                    collect_collector_metrics=not is_3_74_sensor,
                    check_stackrox_logs=True,
                    artifact_destination_prefix=test_versions,
            ),
            # Collection not supported on 3.74
            "pre_test": CollectionMethodOverridePreTest("NO_COLLECTION" if is_3_74_sensor else "core_bpf")
        },
    )
ClusterTestSetsRunner(
    cluster=GKECluster("compat-test",
                       machine_type="e2-standard-8", num_nodes=2),
    initial_pre_test=PreSystemTests(),
    sets=sets,
    final_post=FinalPost(
        store_qa_tests_data=True,
    ),
).run()
