#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import os

from ci_tests import QaE2eTestCompatibility
from compatibility_test import (
    run_compatibility_tests,
)

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# Run supported central and sensor version tuples against QaE2eTestCompatibility (groovy compatibility tests)
run_compatibility_tests(QaE2eTestCompatibility, "compat-test")
