#!/usr/bin/env -S python3 -u

"""
Run qa-tests-backend in a GKE cluster with a `-race` stackrox/main.
"""

import os
import tempfile

from base_qa_e2e_test import make_qa_e2e_test_runner
from clusters import GKECluster

# set test parameters
os.environ["ORCHESTRATOR_FLAVOR"] = "k8s"
os.environ["KUBERNETES_PROVIDER"] = "gke"
os.environ["USE_ROXIE_DEPLOY"] = "true"
os.environ["GCP_IMAGE_TYPE"] = "cos_containerd"
os.environ["SCANNER_V4_DB_STORAGE_CLASS"] = "faster"

# use -rcd image for stackrox/main
os.environ["MAIN_IMAGE_TAG"] = os.environ["STACKROX_BUILD_TAG"] + "-rcd"
os.environ["USE_LOCAL_ROXCTL"] = "true"
os.environ["ROX_RISK_REPROCESSING_INTERVAL"] = "15s"
os.environ["ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL"] = "30s"

with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml") as roxie_config_file:
    main_tag = os.environ["MAIN_IMAGE_TAG"]
    build_tag = os.environ["STACKROX_BUILD_TAG"]
    roxie_config_file.write(f"""\
roxie:
  version: "{build_tag}"
operator:
  operandTagOverrides:
    "main": "{main_tag}"
""")
    roxie_config_file.flush()
    make_qa_e2e_test_runner(
        cluster=GKECluster("race-condition-qa-e2e-test"),
        roxie_config=roxie_config_file.name,
    ).run()
