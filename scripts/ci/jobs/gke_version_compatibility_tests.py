#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import os
from sensor_versions import get_last_sensor_versions_from_git_tags_cli
from compatibility_test import make_compatibility_test_runner
from clusters import GKECluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# don't use postgres
os.environ["ROX_POSTGRES_DATASTORE"] = "false"

versions=get_last_sensor_versions_from_git_tags_cli(4)

gkecluster=GKECluster("qa-e2e-test")

for version in versions:
    os.environ["SENSOR_IMAGE_TAG"] = version
    make_compatibility_test_runner(cluster=gkecluster).run()

print("stub for version compatibility tests")
