#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import logging
from runners import ClusterTestRunner
from clusters import GKECluster

logging.info("Dummy test target gke-nongroovy-comatibility-tests has been called successfully.")

ClusterTestRunner(
    cluster=GKECluster("upgrade-test", machine_type="e2-standard-8")
).run()
