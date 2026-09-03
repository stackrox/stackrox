#!/usr/bin/env -S python3 -u

"""
Run version compatibility tests
"""
import os

from ci_tests import QaE2eGoCompatibilityTest
from compatibility_test import (
    run_compatibility_tests,
)

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["KUBERNETES_PROVIDER"] = "gke"
os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "stackrox-gke-ssd"
# Optimize Scanner V4 startup time by loading required vulnerability bundles
# Nongroovy tests scan RHEL/UBI images (StackRox components) and Ubuntu nodes
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = "rhel-vex,stackrox-rhel-csaf,ubuntu,epss,nvd,manual,osv"

# Run supported central and sensor version tuples against QaE2eGoCompatibilityTest (nongroovy compatibility tests)
run_compatibility_tests(QaE2eGoCompatibilityTest, "nongroovy-compat-test")
