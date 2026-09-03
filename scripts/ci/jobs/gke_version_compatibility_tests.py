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
os.environ["KUBERNETES_PROVIDER"] = "gke"
os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "stackrox-gke-ssd"
# Optimize Scanner V4 startup time by loading only required vulnerability bundles
# Version compat tests scan RHEL/UBI images (StackRox components) and Debian-based GKE infrastructure
os.environ["SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"] = "rhel-vex,debian,manual,epss,nvd,osv"

# Run supported central and sensor version tuples against QaE2eTestCompatibility (groovy compatibility tests)
run_compatibility_tests(QaE2eTestCompatibility, "compat-test")
