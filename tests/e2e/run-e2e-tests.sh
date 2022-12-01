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
 
   Expects a previously configured cluster and runs only selected 
   suite/case. [qa flavor only].
 
Run e2e tests using the working directory code via the rox-ci-image /
stackrox-test container, against the cluster defined in the calling
environment.

Options:
  -c - configure the cluster for test but do not run any tests. [qa flavor only]
  -d - enable debug log gathering to '${QA_TEST_DEBUG_LOGS}'. [qa flavor only]
  -t - override 'make tag' which sets the main version to install and is used by 
       some tests.
  -o - choose the cluster orchestrator. Either k8s or openshift. defaults to k8s.
  -y - run without prompts.
  -h - show this help.

E2e flavor:
  one of qa|e2e|ui|upgrade, defaults to qa

Examples:
# Configure a cluster to run qa-tests-backend/ tests.
$script -c qa

# Run a single qa-tests-backend/ test case (expects a previously configured
# cluster).
$script qa DeploymentTest 'Verify deployment of type Job is deleted once it completes'

# Run the full set of qa-tests-backend/ tests. This is similar to what CI runs
# for *-qa-e2e-tests jobs on a PR.
$script qa

# Run the full set of 'non groovy' e2e tests. This is similar to what CI runs
# for *-nongroovy-e2e-tests jobs on a PR.
$script e2e
_EOH_
    exit 1
}

option_set=":cdhyo:t:"

handle_tag_requirements() {
    while getopts "$option_set" option; do
        case "$option" in
            t)
                export TAG_OVERRIDE="${OPTARG}"
                ;;
            h)
                usage
                ;;
            *)
                ;;
        esac
    done

    tag="$(make tag)"

    if [[ "$tag" =~ -dirty ]]; then
        info "WARN: Dropping -dirty from 'make tag': $tag"
        tag="${tag/-dirty/}"
        export TAG_OVERRIDE="$tag"
    fi

    export ROXCTL_FOR_TEST="$ROOT/bin/linux/roxctl-$tag"

    if [[ ! -f "$ROXCTL_FOR_TEST" ]]; then
        local roxctl_image="quay.io/stackrox-io/roxctl:$tag"
        local id
        id="$(docker create "$roxctl_image")" || {
            cat <<_EOMISSING_
ERROR: Cannot create a container to copy the roxctl binary from: $roxctl_image.
Check that the git commit at $tag was pushed and that that image build/push succeeded.
_EOMISSING_
            exit 1
        }
        docker cp "$id:/roxctl" "$ROXCTL_FOR_TEST"
        docker rm "$id"
    fi
}

if [[ ! -f "/i-am-rox-ci-image" ]]; then
    handle_tag_requirements "$@"
    kubeconfig="${KUBECONFIG:-${HOME}/.kube/config}"
    mkdir -p "${HOME}/.gradle/caches"
    mkdir -p "$QA_TEST_DEBUG_LOGS"
    info "Running in a container..."
    docker run \
      -v "$ROOT:$ROOT:z" \
      -w "$ROOT" \
      -e "KUBECONFIG=${kubeconfig}" \
      -v "${kubeconfig}:${kubeconfig}:z" \
      -v "${HOME}/.gradle/caches:/root/.gradle/caches:z" \
      -v "${GOPATH}/pkg/mod/cache:/go/pkg/mod/cache:z" \
      -v "${QA_TEST_DEBUG_LOGS}:${QA_TEST_DEBUG_LOGS}:z" \
      -e "TAG_OVERRIDE=${TAG_OVERRIDE:-}" \
      -v "${ROXCTL_FOR_TEST}:/usr/local/bin/roxctl:z" \
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

    while getopts "$option_set" option; do
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
            t)
                # handled in the calling context
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

    if [[ "$flavor" == "e2e" ]]; then
        if [[ -n "${suite}" || -n "${case}" ]]; then
            die "ERROR: Suite and Case are not supported with e2e flavor"
        fi
    fi

    # Sanity check that the roxctl in use matches 'make tag'. This should
    # already be true due to the container copy in handle_tag_requirements() but
    # changes to PATH might break that assumption.
    tag="$(make tag)"
    roxctl_version="$(roxctl version)"
    if [[ "${tag}" != "${roxctl_version}" ]]; then
        die "ERROR: 'make tag' and roxctl versions do not match. tag: ${tag} != roxctl: ${roxctl_version}."
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
3. Login to the vault at: https://vault.ci.openshift.org/ui/vault/secrets (Use *OIDC*)
You should see these secrets under kv/
4. Copy a 'token' from that UI and rerun this script.
The 'token' will expire hourly and you will need to renew it through the vault UI.
_EOVAULTHELP_
            exit 1
        fi
    fi

    context="$(kubectl config current-context)"
    echo "This script will tear down resources, install ACS and dependencies and run tests against '$context'."

    if [[ "$prompt" == "true" ]]; then
        read -p "Are you sure? " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Quit."
            exit 1
        fi
    fi

    echo "Importing KV from vault. The following keys will be ignored because they do not match: ^[A-Z]."
    vault kv get -format=json kv/selfservice/stackrox-stackrox-e2e-tests/credentials \
    | jq -r '.data.data | to_entries[] | select( .key|test("^[A-Z]")|not ) | .key'

    eval "$(vault kv get -format=json kv/selfservice/stackrox-stackrox-e2e-tests/credentials \
    | jq -r '.data.data | to_entries[] | select( .key|test("^[A-Z]") ) | "export \(.key|@sh)=\(.value|@sh)"')"

    if ! check_rhacs_eng_image_exists "main" "$tag"; then
        die "ERROR: The main image is not present"
    fi

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
        (
            source "$ROOT/qa-tests-backend/scripts/run-part-1.sh"
            config_part_1
            if [[ "${config_only}" == "false" ]]; then
                test_part_1
            fi
        ) 2>&1 | sed -e 's/^/test output: /'
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
    "$ROOT/tests/e2e/run.sh" 2>&1 | sed -e 's/^/test output: /'
}

main "$@"
