#!/usr/bin/env -S python3 -u

"""
Run compatibility tests against a given cluster.
"""

from pre_tests import PreSystemTests
from ci_tests import QaE2eTestCompatibility
from post_tests import PostClusterTest, FinalPost
from runners import ClusterTestSetsRunner


def make_compatibility_test_runner(cluster):
    return ClusterTestSetsRunner(
        cluster=cluster,
        sets=[
            {
                "name": "version compatibility tests",
                "pre_test": PreSystemTests(),
                "test": QaE2eTestCompatibility(),
                "post_test": PostClusterTest(
                    check_stackrox_logs=True,
                    artifact_destination_prefix="compatibility",
                ),
            },
        ],
        final_post=FinalPost(
            store_qa_tests_data=True,
        ),
    )
