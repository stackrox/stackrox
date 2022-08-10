#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in an OSD AWS cluster provided via automation-flavors/osd.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "openshift"
os.environ["SENSOR_HELM_DEPLOY"] = "true"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
