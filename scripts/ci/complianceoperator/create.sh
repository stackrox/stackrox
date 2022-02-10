#! /bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

echo "Deploying Compliance Operator"
git clone git@github.com:openshift/compliance-operator.git
cd compliance-operator

oc create -f deploy/ns.yaml
for f in $(ls -1 deploy/crds/*crd.yaml); do oc apply -f $f -n openshift-compliance; done
oc apply -n openshift-compliance -f deploy/

# Due to reconciliation bug in compliance operator, ensure that the profile exists prior to creating the
# tailored profile
success=0
profile="rhcos4-moderate"
for i in {1..120}; do
  if [[ $(oc -n openshift-compliance get profiles.compliance $profile -o name | awk 'END{print NR}') == 1 ]]; then
    success=1
    break
  fi
  sleep 5
  echo "Currently have waited $((i * 5)) for profile $profile to exist"
done

if [[ "${success}" == 0 ]]; then
  echo "Failed to get profile $profile. Expect some compliance tests to fail"
fi

function wait_for_rule {
  rule=$1
  success=0
  for i in {1..120}; do
    if [[ $(oc -n openshift-compliance get rules.compliance $rule -o name | awk 'END{print NR}') == 1 ]]; then
      success=1
      break
    fi
    sleep 5
    echo "Currently have waited $((i * 5)) for rule $rule to exist"
  done

  if [[ "${success}" == 0 ]]; then
    echo "Failed to get rule $rule. Expect some compliance tests to fail"
  fi
}

wait_for_rule "rhcos4-usbguard-allow-hid-and-hub"
wait_for_rule "rhcos4-zipl-page-poison-argument"

echo "Deploying tailored profiles"
oc -n openshift-compliance create -R -f "${DIR}/tailoredprofile"
echo "Deploying scan setting bindings"
oc -n openshift-compliance create -R -f "${DIR}/ssb"
