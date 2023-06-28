#! /bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

echo "Deploying Compliance Operator"
git clone git@github.com:ComplianceAsCode/compliance-operator.git
cd compliance-operator

# Install the Compliance Operator through its own tooling. This helps simplify
# the installation process, so that what we're using here doesn't diverge from
# what the Compliance Operator does. Specifically, this builds the container
# images based on the latest source and uploads them to the image registry
# available in the cluster before installing the operator. This requires that
# you've authenticated to the cluster using `oc login` and is
# OpenShift-specific.
make deploy-local


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
