#!/usr/bin/env bash
# shellcheck disable=SC1091

# Run a full suite of e2e tests using the code via the rox-ci-image / stackrox-test container,
# against the cluster defined in the calling environment.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/lib.sh"
source "$ROOT/scripts/ci/lib.sh"
source "$ROOT/scripts/ci/gcp.sh"

usage() { 
    cat <<_EOH_
Usage: $0 [-m <tag>] [-o k8s|openshift] [-y] [<e2e flavor: one of qa|e2e|ui|upgrade, defaults to qa>]
  -m - override 'make tag' for the version to install.
  -o - choose the cluster variety. defaults to k8s.
  -y - run without prompts.
_EOH_
    exit 1
}

if [[ ! -f "/i-am-rox-ci-image" ]]; then
    kubeconfig="${KUBECONFIG:-${HOME}/.kube/config}"
    docker run \
      -v "$ROOT:$ROOT:z" \
      -w "$ROOT" \
      -e "KUBECONFIG=${kubeconfig}" \
      -v "${kubeconfig}:${kubeconfig}:z" \
      -e VAULT_TOKEN \
      --platform linux/amd64 \
      --rm -it \
      --entrypoint="$0" \
      quay.io/stackrox-io/apollo-ci:stackrox-test-0.3.53 "$@"
    exit 0
fi

orchestrator="k8s"
prompt="true"

while getopts ":yo:m:" option; do
    case "$option" in
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
shift $((OPTIND-1))

flavor="${1:-qa}"
case "$flavor" in
    qa)
        ;;
    *)
        die "flavor $flavor not supported"
        ;;
esac

case "$orchestrator" in
    k8s|openshift)
        ;;
    *)
        usage
        ;;
esac

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

export PATH="${ROOT}/bin/linux_amd64:${PATH}"
main_version="${MAIN_IMAGE_TAG:-$tag}"
roxctl_version="$(roxctl version)"
if [[ "${main_version}" != "${roxctl_version}" ]]; then
    cat <<_EOVERSION_
ERROR: main and roxctl versions do not match. main: ${main_version} != roxctl: ${roxctl_version}.
They are required to match for the ./deploy scripts to run without docker.
Run 'make cli' to get matching versions.
_EOVERSION_
    exit 1
fi

# Do we need to login to vault?
export VAULT_ADDR=https://vault.ci.openshift.org/
if ! vault kv list kv/selfservice/stackrox-stackrox-e2e-tests > /dev/null 2>&1; then
    echo "Login to OpenShift CI Vault."
    vault login || true

    if ! vault kv list kv/selfservice/stackrox-stackrox-e2e-tests 2>&1 | sed -e 's/^/vault output: /'; then
        cat <<_EOVAULTHELP_
ERROR: Cannot list vault secrets.
There are a number of required steps to get access to vault:
1. Log in to the secrets collection manager at https://selfservice.vault.ci.openshift.org/secretcollection?ui=true
(This is a Red Hat-ism and will require SSO)
2. Ask a team member to add you to the collections required for this test:
stackrox-stackrox-initial and stackrox-stackrox-e2e-tests.
3. Login to the vault at: https://vault.ci.openshift.org/ui/vault/secrets (Use OIDC)
You should see these secrets under kv/
4. Copy a 'token' from that UI and rerun this script.
The 'token' will expire periodically and you will need to renew it through the vault UI.
_EOVAULTHELP_
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

# GCP login using the CI service account is required to access infra GKE clusters.
setup_gcp

if ! kubectl get nodes > /dev/null; then
    die "ERROR: Cannot access a cluster in your environment. Check KUBECONFIG, etc"
fi

info "Running the test."

export ORCHESTRATOR_FLAVOR="$orchestrator"

# required to get a running central
export ROX_POSTGRES_DATASTORE="${ROX_POSTGRES_DATASTORE:-false}"

"$ROOT/qa-tests-backend/scripts/run-part-1.sh" 2>&1 | sed -e 's/^/test output: /'
