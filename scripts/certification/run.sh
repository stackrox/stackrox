#!/bin/bash

set -eo pipefail

usage() {
  echo "usage: $0 <ENDPOINT> [<CENTRAL_PASSPHRASE>]"
  exit 2
}

ENDPOINT=$1
PASSPHRASE=$2
if [[ -z $ENDPOINT ]]; then
  usage
fi
if [[ -z $ROX_API_TOKEN && -z $PASSPHRASE ]]; then
  usage
fi

function cli() {
  if [[ -z $ROX_API_TOKEN ]]; then
    roxctl --ca "" --insecure-skip-tls-verify -e "$ENDPOINT" -p "$PASSPHRASE" "$@"
  else
    roxctl --ca "" --insecure-skip-tls-verify -e "$ENDPOINT" "$@"
  fi
}

function jq_csv() {
  jq -r '.scan.components[]? | { name: .name, version: .version, vulns: .vulns[]? | { cve: .cve, cvss: .cvss, fixedBy: .fixedBy, link: .link, severity: .severity } | flatten } | flatten | @csv'
}

images=("stackrox/sandbox:nodejs-10" "stackrox/sandbox:jenkins-agent-maven-35-rhel7")
names=("nodejs" "jenkins")

i=0
for img in ${images[@]}; do
  name=${names[i]}
  cli image scan -i $img | jq_csv > output.csv
  python3 format_csv.py output.csv "$name".csv
  rm output.csv
  i=$((i+1))
done
