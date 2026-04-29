#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in an AKS cluster provided via automation-flavors/aks.
"""
import os
from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import AutomationFlavorsCluster

# set required test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["KUBERNETES_PROVIDER"] = "aks"

os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

# This test suite has been migrated to use roxie for deployment (deploy_stackrox_with_roxie_compat()) instead of
# the legacy deployment flow (deploy_stackrox()).
#
# The previous deployment mechanism used environment variables extensively for deployment configuration.
# These variables were injected into deployment manifests and/or translated into roxctl command-line arguments
# in multiple places, which makes the whole configuration setup difficult to maintain and reason about.
#
# The compatibility layer for roxie-based deployments (deploy_stackrox_with_roxie_compat()) is designed as a
# drop-in replacement for the legacy deployment mechanism (deploy_stackrox()) and picks up the same environment
# variables for configuration with the same defaulting behaviour.
#
# Long term, the goal is to migrate all test suites to use the modern roxie-based deployment mechanism,
# where the entire deployment configuration is to be assembled explicitly in a YAML configuration file.
os.environ["USE_ROXIE_DEPLOY"] = "true"

make_qa_e2e_test_runner(cluster=AutomationFlavorsCluster()).run()
