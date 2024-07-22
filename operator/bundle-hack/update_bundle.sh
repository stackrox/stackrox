#!/usr/bin/env bash

export STACKROX_MAIN_IMAGE_PULLSPEC="quay.io/rhacs-eng/main@sha256:cbb656d5167ceb65a0127f7d9fd2fc0eaf932e3fd127f0047dfd29e7f0d1990f"

export STACKROX_OPERATOR_IMAGE_PULLSPEC="quay.io/rhacs-eng/stackrox-operator@sha256:cbb656d5167ceb65a0127f7d9fd2fc0eaf932e3fd127f0047dfd29e7f0d1990f"

export CSV_FILE=/manifests/gatekeeper-operator.clusterserviceversion.yaml

sed -i -e "s|quay.io/rhacs-eng/main:v.*|\"${STACKROX_MAIN_IMAGE_PULLSPEC}\"|g" \
	-e "s|quay.io/rhacs-eng/rhacs-operator:v.*|\"${STACKROX_OPERATOR_IMAGE_PULLSPEC}\"|g" \
	"${CSV_FILE}"

AMD64_BUILT=$(skopeo inspect --raw docker://${STACKROX_OPERATOR_IMAGE_PULLSPEC} | jq -e '.manifests[] | select(.platform.architecture=="amd64")')
export AMD64_BUILT
ARM64_BUILT=$(skopeo inspect --raw docker://${STACKROX_OPERATOR_IMAGE_PULLSPEC} | jq -e '.manifests[] | select(.platform.architecture=="arm64")')
export ARM64_BUILT

# TODO: Should these builds be enabled as well?
# PPC64LE_BUILT=$(skopeo inspect --raw docker://${STACKROX_OPERATOR_IMAGE_PULLSPEC} | jq -e '.manifests[] | select(.platform.architecture=="ppc64le")')
# export PPC64LE_BUILT
# S390X_BUILT=$(skopeo inspect --raw docker://${STACKROX_OPERATOR_IMAGE_PULLSPEC} | jq -e '.manifests[] | select(.platform.architecture=="s390x")')
# export S390X_BUILT

EPOC_TIMESTAMP=$(date +%s)
export EPOC_TIMESTAMP
# time for some direct modifications to the csv
python3 - << CSV_UPDATE
import os
from collections import OrderedDict
from sys import exit as sys_exit
from datetime import datetime
from ruamel.yaml import YAML
yaml = YAML()
def load_manifest(pathn):
   if not pathn.endswith(".yaml"):
      return None
   try:
      with open(pathn, "r") as f:
         return yaml.load(f)
   except FileNotFoundError:
      print("File can not found")
      exit(2)

def dump_manifest(pathn, manifest):
   with open(pathn, "w") as f:
      yaml.dump(manifest, f)
   return
timestamp = int(os.getenv('EPOC_TIMESTAMP'))
datetime_time = datetime.fromtimestamp(timestamp)
stackrox_csv = load_manifest(os.getenv('CSV_FILE'))
# Add arch support labels
stackrox_csv['metadata']['labels'] = stackrox_csv['metadata'].get('labels', {})
if os.getenv('AMD64_BUILT'):
	stackrox_csv['metadata']['labels']['operatorframework.io/arch.amd64'] = 'supported'
if os.getenv('ARM64_BUILT'):
	stackrox_csv['metadata']['labels']['operatorframework.io/arch.arm64'] = 'supported'
if os.getenv('PPC64LE_BUILT'):
	stackrox_csv['metadata']['labels']['operatorframework.io/arch.ppc64le'] = 'supported'
if os.getenv('S390X_BUILT'):
	stackrox_csv['metadata']['labels']['operatorframework.io/arch.s390x'] = 'supported'
stackrox_csv['metadata']['labels']['operatorframework.io/os.linux'] = 'supported'
stackrox_csv['metadata']['annotations']['createdAt'] = datetime_time.strftime('%d %b %Y, %H:%M')
stackrox_csv['metadata']['annotations']['features.operators.openshift.io/disconnected'] = 'true'
stackrox_csv['metadata']['annotations']['features.operators.openshift.io/fips-compliant'] = 'true'
stackrox_csv['metadata']['annotations']['features.operators.openshift.io/proxy-aware'] = 'false'
stackrox_csv['metadata']['annotations']['features.operators.openshift.io/tls-profiles'] = 'false'
stackrox_csv['metadata']['annotations']['features.operators.openshift.io/token-auth-aws'] = 'false'
stackrox_csv['metadata']['annotations']['features.operators.openshift.io/token-auth-azure'] = 'false'
stackrox_csv['metadata']['annotations']['features.operators.openshift.io/token-auth-gcp'] = 'false'
stackrox_csv['metadata']['annotations']['repository'] = 'https://github.com/stackrox/stackrox'
stackrox_csv['metadata']['annotations']['containerImage'] = os.getenv('STACKROX_OPERATOR_IMAGE_PULLSPEC', '')

dump_manifest(os.getenv('CSV_FILE'), stackrox_csv)
CSV_UPDATE

cat $CSV_FILE
