#!/usr/bin/env -S python3 -u

"""
Runs version compatibility tests against the supplied testfunc
"""
import logging
import os
import sys

from pre_tests import (
    PreSystemTests,
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

    # Get the test tuples (central_version, sensor_version) for supported versions
    test_tuples = get_compatibility_test_tuples()

    if len(test_tuples) > 0:
        sets = []
        for test_tuple in test_tuples:
            os.environ["ROX_TELEMETRY_STORAGE_KEY_V1"] = 'DISABLED'
            test_versions = f'{test_tuple.central_version}--{test_tuple.sensor_version}'

            logging.info("Running compatibility tests for central-v%s, sensor-v%s with function %s",
                         test_tuple.central_version, test_tuple.sensor_version, testfunc.__name__)

            sets.append(
                TestSet(
                    f'version compatibility tests: {test_versions}',
                    test=testfunc(test_tuple.central_version, test_tuple.sensor_version),
                    post=PostClusterTest(
                        check_stackrox_logs=True,
                        artifact_destination_prefix=test_versions,
                    )
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
