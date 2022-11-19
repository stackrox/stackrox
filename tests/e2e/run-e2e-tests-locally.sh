#!/usr/bin/env bash
# shellcheck disable=SC1091

# Run a full suite of e2e tests using the code and against the cluster defined
# in the calling environment.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/lib.sh"

usage() { 
    cat <<_EOH_
Usage: $0  [-d] [-m <tag>] [-o k8s|openshift] [-y]
  -d - allow docker to run roxctl.
  -m - override 'make tag' for the version to install.
  -o - choose the cluster variety. defaults to k8s.
  -y - run without prompts.
_EOH_
    exit 1
}

allow_docker="false"
orchestrator="k8s"
prompt="true"

while getopts ":dyo:m:" option; do
    case "$option" in
        d)
            allow_docker="true"
            ;;
        o)
            orchestrator="${OPTARG}"
            ;;
        m)
            export MAIN_IMAGE_TAG="${OPTARG}"
            ;;
        y)
            prompt="false"
            ;;
        *)
            usage
            ;;
    esac
done

case "$orchestrator" in
    k8s|openshift)
        ;;
    *)
        usage
        ;;
esac

### Environment checks - Are we ready?

info "Starting environment checks."

if ! kubectl get nodes > /dev/null; then
    die "ERROR: Cannot access a cluster in your environment. Check KUBECONFIG, etc"
fi

if ! command -v vault > /dev/null 2>&1; then
    if [[ -f "/etc/redhat-release" ]]; then
        cat <<_EOV_
ERROR: There is no hashicorp 'vault' command in your path.
For RHEL/Centos try:
  sudo dnf config-manager --add-repo https://rpm.releases.hashicorp.com/RHEL/hashicorp.repo
  sudo dnf install -y vault
_EOV_
        exit 1
    else
        die "ERROR: There is no hashicorp 'vault' command in your path. See https://developer.hashicorp.com/vault/downloads"
    fi
fi

export VAULT_ADDR=https://vault.ci.openshift.org/

if ! vault kv list kv/selfservice/stackrox-stackrox-e2e-tests 2>&1 | sed -e 's/^/vault output: /'; then
    cat <<_EOL_
ERROR: Cannot list the vault secrets.
There are a number of required steps:
1. Log in to the secrets collection manager at https://selfservice.vault.ci.openshift.org/secretcollection?ui=true
   (This is a RedHat-ism and will require SSO)
2. Ask a team member to add you to the collections required for this test.
   stackrox-stackrox-initial and stackrox-stackrox-e2e-tests
3. Login to the vault at: https://vault.ci.openshift.org/ui/vault/secrets
   You should see the secrets under kv/
4. Copy a 'token' from that UI and use it to log in to vault with:
   export VAULT_ADDR=https://vault.ci.openshift.org/
   vault login
5. Test with:
   vault kv list kv/selfservice/stackrox-stackrox-e2e-tests
6. Rerun this script.
The 'token' will expire periodically and you will need to re-login.
_EOL_
    exit 1
fi

### Prep for test

info "Preparing for test."

cd "$ROOT"

tag="$(make tag)"
if [[ -z "${MAIN_IMAGE_TAG:-}" && "$tag" =~ -dirty ]]; then
    cat <<_EODIRTY_
ERROR: The tag for the working directory includes a '-dirty' tag. 
It is unlikely that that has been pushed to registries. Specify a
valid tag with -m or set MAIN_IMAGE_TAG.
_EODIRTY_
    exit 1
fi

if [[ "$allow_docker" == "false" ]]; then
    main_version="${MAIN_IMAGE_TAG:-$tag}"
    roxctl_version="$(roxctl version)"
    if [[ "${main_version}" != "${roxctl_version}" ]]; then
        cat <<_EOVERSION_
ERROR: main and roxctl versions do not match. main: ${main_version} != roxctl: ${roxctl_version}.
They must match for the ./deploy scripts to run without docker.
Use '-d' to run roxctl via docker. This works under MacOS at least.
Or 'make cli' to get matching versions.
_EOVERSION_
        exit 1
    fi
fi

context="$(kubectl config current-context)"
echo "This script will tear down resources, install ACS and run tests against '$context'."

if [[ "$prompt" == "true" ]]; then
    read -p "Are you sure? " -r
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Quit."
        exit 1
    fi
fi

# TODO got image? - for a 'full' test run we might want to poll for 'canonical'
# images - poll_for_system_test_images

echo "Importing KV from vault. The following keys will be ignored because they do not match: ^[A-Z]."
vault kv get -format=json kv/selfservice/stackrox-stackrox-e2e-tests/credentials \
  | jq -r '.data.data | to_entries[] | select( .key|test("^[A-Z]")|not ) | .key'

eval "$(vault kv get -format=json kv/selfservice/stackrox-stackrox-e2e-tests/credentials \
  | jq -r '.data.data | to_entries[] | select( .key|test("^[A-Z]") ) | "export \(.key|@sh)=\(.value|@sh)"')"

### Run the test! Hold on to your hats.

info "Running the test."

export ORCHESTRATOR_FLAVOR="$orchestrator"

# Using helm in this manner saves the ./deploy scripts trying to run roxctl via
# docker which can be very system dependent.
export OUTPUT_FORMAT="helm"

# --flavor qa
"$ROOT/qa-tests-backend/scripts/run-part-1.sh" 2>&1 | sed -e 's/^/test output: /'
# --flavor e2e
#"$ROOT/tests/e2e/run.sh"
