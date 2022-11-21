#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import os
from get_latest_release_versions import get_latest_release_versions
from compatibility_test import make_compatibility_test_runner
from clusters import GKECluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# don't use postgres
os.environ["ROX_POSTGRES_DATASTORE"] = "false"

# versions=get_latest_release_versions(4)
chart_versions=["72.1.0", "71.2.0", "70.0.0"]

gkecluster=GKECluster("compat-test")

for version in chart_versions:
    os.environ["SENSOR_CHART_VERSION"] = version
    make_compatibility_test_runner(cluster=gkecluster).run()
