#!/usr/bin/env -S python3 -u

"""
Run tests/e2e in a OCP cluster
"""
import os
from runners import ClusterTestRunner
from pre_tests import PreSystemTests
from ci_tests import SensorIntegration

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"
os.environ["ROX_ACTIVE_VULN_MGMT"] = "true"

ClusterTestRunner(
    pre_test=PreSystemTests(run_poll_for_system_test_images=False),
    test=SensorIntegration(),
).run()
