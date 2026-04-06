#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import os
import scanner_v4_defaults
from ci_tests import QaE2eTestCompatibility
from compatibility_test import (
    run_compatibility_tests,
)

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"

# Scanner V4
os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "stackrox-gke-ssd"
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = scanner_v4_defaults.VULN_BUNDLE_ALLOWLIST

# Run supported central and sensor version tuples against QaE2eTestCompatibility (groovy compatibility tests)
run_compatibility_tests(QaE2eTestCompatibility, "compat-test")
