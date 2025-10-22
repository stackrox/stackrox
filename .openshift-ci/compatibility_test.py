#!/usr/bin/env -S python3 -u

"""
Runs version compatibility tests against the supplied testfunc
"""
import logging
import os
import sys

from pre_tests import (
    PreSystemTests,
    CollectionMethodOverridePreTest
)
from post_tests import PostClusterTest, FinalPost
from runners import ClusterTestSetsRunner, TestSet
from clusters import GKECluster
from get_compatibility_test_tuples import (
    get_compatibility_test_tuples,
)


def run_compatibility_tests(testfunc, cluster_name):
    # start logging
    logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

    # Get the test tuples (central_version, sensor_version) for supported versions with available helm charts
    test_tuples = get_compatibility_test_tuples()

    if len(test_tuples) > 0:
        sets = []
        for test_tuple in test_tuples:
            os.environ["ROX_TELEMETRY_STORAGE_KEY_V1"] = 'DISABLED'
            test_versions = f'{test_tuple.central_version}--{test_tuple.sensor_version}'

            # expected version string is like 74.x.x for ACS 3.74 versions
            is_3_74_sensor = test_tuple.sensor_version.startswith('74')

            logging.info("Running compatibility tests for central-v%s, sensor-v%s with function %s",
                         test_tuple.central_version, test_tuple.sensor_version, testfunc.__name__)

            sets.append(
                TestSet(
                    f'version compatibility tests: {test_versions}',
                    test=testfunc(test_tuple.central_version, test_tuple.sensor_version),
                    post=PostClusterTest(
                        collect_collector_metrics=not is_3_74_sensor,
                        check_stackrox_logs=True,
                        artifact_destination_prefix=test_versions,
                    ),
                    # Collection not supported on 3.74
                    pre=CollectionMethodOverridePreTest("NO_COLLECTION" if is_3_74_sensor else "core_bpf")
                )
            )
        ClusterTestSetsRunner(
            cluster=GKECluster(cluster_name,
                               machine_type="e2-standard-8", num_nodes=2),
            initial_pre_test=PreSystemTests(),
            sets=sets,
            final_post=FinalPost(
                store_qa_tests_data=True,
            ),
        ).run()
    else:
        logging.info("There are currently no supported older versions or support exceptions that require compatibility "
                     "testing.")
