#!/usr/bin/env bash
# shellcheck disable=SC1091

# Run e2e tests using the working directory code via the rox-ci-image /
# stackrox-test container, against the cluster defined in the calling
# environment.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/lib.sh"
source "$ROOT/scripts/ci/lib.sh"
source "$ROOT/scripts/ci/gcp.sh"
source "$ROOT/tests/e2e/lib.sh"

require_environment "QA_TEST_DEBUG_LOGS"

usage() {
    script=$(basename "$0")
    cat <<_EOH_
Usage:
 $script [Options...] [E2e flavor]
 
   Configures the cluster and runs all suites.
 
 $script [Options...] [E2e flavor] Suite [Case]
 
   Expects a previously configured cluster and runs only selected suite/case.
 

Run e2e tests using the working directory code via the rox-ci-image /
stackrox-test container, against the cluster defined in the calling
environment.

Options:
  -c - configure the cluster for test but do not run any tests.
  -d - enable debug log gathering to '${QA_TEST_DEBUG_LOGS}'.
  -m - override 'make tag' for the version to install.
  -o - choose the cluster variety. defaults to k8s.
  -y - run without prompts.

E2e flavor:
  one of qa|e2e|ui|upgrade, defaults to qa

Examples:
# Configure a cluster to run qa-tests-backend/ tests.
$script -c qa

# Run a single qa-tests-backend/ test case (expects a previously configured
# cluster).
$script qa DeploymentTest 'Verify deployment of type Job is deleted once it completes'

# Run the full set of qa-tests-backend/ tests. This is similar to what CI runs
# for a PR as *-qa-e2e-tests.
$script qa

# Run the full set of tests/ e2e tests. This is similar to what CI runs
# for a PR as *-nongroovy-e2e-tests.
$script e2e
_EOH_
    exit 1
}

if [[ ! -f "/i-am-rox-ci-image" ]]; then
    kubeconfig="${KUBECONFIG:-${HOME}/.kube/config}"
    mkdir -p "${HOME}/.gradle/caches"
    mkdir -p "$QA_TEST_DEBUG_LOGS"
    docker run \
      -v "$ROOT:$ROOT:z" \
      -w "$ROOT" \
      -e "KUBECONFIG=${kubeconfig}" \
      -v "${kubeconfig}:${kubeconfig}:z" \
      -v "${HOME}/.gradle/caches:/root/.gradle/caches:z" \
      -v "${GOPATH}/pkg/mod/cache:/go/pkg/mod/cache:z" \
      -v "${QA_TEST_DEBUG_LOGS}:${QA_TEST_DEBUG_LOGS}:z" \
      -e VAULT_TOKEN \
      --platform linux/amd64 \
      --rm -it \
      --entrypoint="$0" \
      quay.io/stackrox-io/apollo-ci:stackrox-test-0.3.53 "$@"
    exit 0
fi

main() {
    config_only="false"
    orchestrator="k8s"
    prompt="true"

    while getopts ":cdyo:m:" option; do
        case "$option" in
            c)
                config_only="true"
                ;;
            d)
                export GATHER_QA_TEST_DEBUG_LOGS="true"
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
    shift $((OPTIND-1))

    flavor="${1:-qa}"
    case "$flavor" in
        qa|e2e)
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

    suite="${2:-}"
    case="${3:-}"

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

    case "$flavor" in
        qa)
            run_qa_flavor
            ;;
        e2e)
            run_e2e_flavor
            ;;
        *)
            die "flavor $flavor not supported"
            ;;
    esac
}

run_qa_flavor() {
    if [[ -z "$suite" && -z "$case" ]]; then
        source "$ROOT/qa-tests-backend/scripts/run-part-1.sh"
        config_part_1 2>&1 | sed -e 's/^/config output: /'
        if [[ "${config_only}" == "false" ]]; then
            test_part_1 2>&1 | sed -e 's/^/test output: /'
        fi
    else
        export_test_environment
        setup_deployment_env false false
        export CLUSTER="${ORCHESTRATOR_FLAVOR^^}"
        wait_for_api
        export DEPLOY_DIR="$ROOT/deploy/${ORCHESTRATOR_FLAVOR}"
        get_central_basic_auth_creds

        pushd qa-tests-backend
        if [[ -z "$case" ]]; then
            ./gradlew test --console=plain --tests="$suite"
        else
            ./gradlew test --console=plain --tests="$suite.$case"
        fi
        popd
    fi
}

run_e2e_flavor() {
    "$ROOT/tests/e2e/run.sh" | sed -e 's/^/test output: /'
}

main "$@"
