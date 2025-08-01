#!/usr/bin/env -S python3 -u

import os
from runners import ClusterTestRunner
from pre_tests import PreSystemTests
from ci_tests import SensorProfilingOCP

# set required test parameters
os.environ["DEPLOY_STACKROX_VIA_OPERATOR"] = "true"
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["ROX_POSTGRES_DATASTORE"] = "true"

ClusterTestRunner(
    pre_test=PreSystemTests(run_poll_for_system_test_images=False),
    test=SensorProfilingOCP(),
).run()
