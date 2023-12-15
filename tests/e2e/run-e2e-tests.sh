#!/usr/bin/env bash

# Run e2e tests using the working directory code via the rox-ci-image /
# stackrox-test container, against the cluster defined in the calling
# environment.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$ROOT/scripts/lib.sh"
# shellcheck source=../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/gcp.sh
source "$ROOT/scripts/ci/gcp.sh"
# shellcheck source=../../tests/e2e/lib.sh
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
  -c, --config-only - configure the cluster for test but do not run
    any tests. [qa flavor only]
  --test-only - reuse prior configuration and run tests. [qa flavor
    only]
  -d, --gather-debug - enable debug log gathering to '${QA_TEST_DEBUG_LOGS}'.
    [qa flavor only]
  -s, --spin-cycle=<count> - repeat the test portion until a failure
    occurs or <count> is reached with no failures. [qa flavor only]
  -w, --spin-wait=<seconds> - delay between tests when running repeat 
    tests. default: no wait. [qa flavor only]
  -t <tag> - override 'make tag' which sets the main version to install
    and is used by some tests.
  -o, --orchestrator=<orchestrator> - choose the cluster orchestrator.
    Either k8s or openshift. defaults to k8s.
  --db=<postgres|rocksdb> - defaults to postgres.
  -y - run without prompts.
  -h - show this help.

E2e flavor:
  one of qa|e2e, defaults to qa

Examples:
# Configure a cluster to run qa-tests-backend/ tests.
$script --config-only qa

# Run a single qa-tests-backend/ test case (expects a previously configured
# cluster).
$script qa DeploymentTest 'Verify deployment of type Job is deleted once it completes'

# Run the full set of qa-tests-backend/ tests. This is similar to what CI runs
# for *-qa-e2e-tests jobs on a PR.
$script qa

# Run the full set of 'non groovy' e2e tests. This is similar to what CI runs
# for *-nongroovy-e2e-tests jobs on a PR.
$script e2e

For more details see tests/e2e/run-e2e-tests-README.md.
_EOH_
    exit 1
}

handle_tag_requirements() {
    get_initial_options "$@"

    tag="$(make tag)"

    if [[ "$tag" =~ -dirty ]]; then
        info "WARN: Dropping -dirty from 'make tag': $tag"
        tag="${tag/-dirty/}"
        export BUILD_TAG="$tag"
    fi

    export ROXCTL_FOR_TEST="$ROOT/bin/linux/roxctl-$tag"
    mkdir -p "$ROOT/bin/linux"

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

get_initial_options() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -t)
                export BUILD_TAG="$2"
                shift 2
                ;;
            -h)
                usage
                ;;
            *)
                shift
                ;;
        esac
    done
}

if [[ ! -f "/i-am-rox-ci-image" ]]; then
    handle_tag_requirements "$@"
    kubeconfig="${KUBECONFIG:-${HOME}/.kube/config}"
    mkdir -p "$QA_TEST_DEBUG_LOGS"
    info "Running in a container..."
    docker run \
      -v "$ROOT:$ROOT:z" \
      -w "$ROOT" \
      -e "KUBECONFIG=${kubeconfig}" \
      -v "${kubeconfig}:${kubeconfig}:z" \
      -v "${GOPATH}/pkg/mod/cache:/go/pkg/mod/cache:z" \
      -v "${QA_TEST_DEBUG_LOGS}:${QA_TEST_DEBUG_LOGS}:z" \
      -e "BUILD_TAG=${BUILD_TAG:-}" \
      -v "${ROXCTL_FOR_TEST}:/usr/local/bin/roxctl:z" \
      -e VAULT_TOKEN \
      --platform linux/amd64 \
      --rm -it \
      --entrypoint="$0" \
      quay.io/stackrox-io/apollo-ci:stackrox-test-0.3.59 "$@"
    exit 0
fi

get_options() {
    # in stackrox-test container getopt supports long options
    normalized_opts=$(\
      getopt \
        -o cdo:s:w:t:y \
        --long config-only,test-only,gather-debug,spin-cycle:,spin-wait:,orchestrator:,db: \
        -n 'run-e2e-tests.sh' -- "$@" \
    )

    eval set -- "$normalized_opts"

    export CONFIG_ONLY="false"
    export TEST_ONLY="false"
    export GATHER_QA_TEST_DEBUG_LOGS="false"
    export SPIN_CYCLE_COUNT=1
    export SPIN_CYCLE_WAIT=0
    export ORCHESTRATOR="k8s"
    export DATABASE="postgres"
    export PROMPT="true"

    while true; do
        case "$1" in
            -c | --config-only)
                export CONFIG_ONLY="true"
                shift
                ;;
            --test-only)
                export TEST_ONLY="true"
                shift
                ;;
            -d | --gather-debug)
                export GATHER_QA_TEST_DEBUG_LOGS="true"
                shift
                ;;
            -s | --spin-cycle)
                export SPIN_CYCLE_COUNT="$2"
                shift 2
                ;;
            -w | --spin-wait)
                export SPIN_CYCLE_WAIT="$2"
                shift 2
                ;;
            -o | --orchestrator)
                export ORCHESTRATOR="$2"
                shift 2
                ;;
            --db)
                export DATABASE="$2"
                shift 2
                ;;
            -t)
                # handled in the calling context
                shift 2
                ;;
            -y)
                export PROMPT="false"
                shift
                ;;
            --)
                shift
                break
                ;;
        esac
    done

    export FLAVOR="${1:-qa}"
    case "$FLAVOR" in
        qa|e2e)
            ;;
        *)
            die "flavor $FLAVOR not supported"
            ;;
    esac
    export ROX_POSTGRES_DATASTORE="true"
    export TASK_OR_SUITE="${2:-}"
    export CASE="${3:-}"

    export_job_name

    if [[ "${CONFIG_ONLY}" == "true" && "${TEST_ONLY}" == "true" ]]; then
        die "--config-only and --test-only are mutually exclusive"
    fi

    if [[ "$FLAVOR" == "e2e" ]]; then
        if [[ -n "${TASK_OR_SUITE}" || -n "${CASE}" ]]; then
            die "ERROR: Target, Suite and Case are not supported with e2e flavor"
        fi
        if [[ "${CONFIG_ONLY}" == "true" || "${TEST_ONLY}" == "true" ]]; then
            die "--config-only and --test-only are not supported with e2e flavor"
        fi
    fi
}

main() {
    get_options "$@"

    case "$ORCHESTRATOR" in
        k8s|openshift)
            ;;
        *)
            usage
            ;;
    esac

    cd "$ROOT"

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
    cat <<_EOWARNING_
WARNING! This script can be destructive. Depending on how it is invoked,
it may tear down resources, install ACS and dependencies and run tests.
Current cluster context is: '$context'.
_EOWARNING_
    if [[ "$PROMPT" == "true" ]]; then
        read -p "Are you sure? " -r
        if [[ ! $REPLY =~ ^[Yy]e?s?$ ]]; then
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

    export ORCHESTRATOR_FLAVOR="$ORCHESTRATOR"

    # required to get a running central
    export ROX_POSTGRES_DATASTORE="true"

    case "$FLAVOR" in
        qa)
            run_qa_flavor
            ;;
        e2e)
            run_e2e_flavor
            ;;
        *)
            die "flavor $FLAVOR not supported"
            ;;
    esac
}

run_qa_flavor() {
    source "$ROOT/qa-tests-backend/scripts/run-part-1.sh"
    setup_podsecuritypolicies_config

    if [[ -z "${TASK_OR_SUITE}" && -z "${CASE}" ]]; then
        (
            if [[ "${TEST_ONLY}" == "false" ]]; then
                config_part_1
                info "Config succeeded."
            else
                reuse_config_part_1
                info "Config reuse succeeded."
            fi
            if [[ "${CONFIG_ONLY}" == "false" ]]; then
                spin test_part_1
                info "Test succeeded."
            fi
        ) 2>&1 | sed -e 's/^/test output: /'
    else
        reuse_config_part_1
        info "Config reuse succeeded."

        pushd qa-tests-backend
        if [[ -z "${CASE}" ]]; then
            if [[ "${TASK_OR_SUITE}" =~ ^[A-Z] ]]; then
                # Suite (.groovy test Specification)
                spin ./gradlew test --console=plain --tests="${TASK_OR_SUITE}"
            else
                # build.gradle task
                spin ./gradlew "${TASK_OR_SUITE}" --console=plain
            fi
        else
            spin ./gradlew test --console=plain --tests="${TASK_OR_SUITE}.${CASE}"
        fi
        popd
    fi
}

run_e2e_flavor() {
    "$ROOT/tests/e2e/run.sh" 2>&1 | sed -e 's/^/test output: /'
}

spin() {
    local count=0
    while (( SPIN_CYCLE_COUNT > count )); do
        "$@"
        (( count++ )) || true
        info "Completed test cycle: $count"
        if (( SPIN_CYCLE_COUNT > count )); then
            info "Waiting ${SPIN_CYCLE_WAIT} seconds between test cycles to allow resources to complete deletion"
            sleep "${SPIN_CYCLE_WAIT}"
        fi
    done
}

export_job_name() {
    # Emulate CI_JOB_NAME (which sets Env.CI_JOB_NAME for .groovy tests) as it is
    # used to determine some test behavior.
    local job_name=""

    case "$FLAVOR" in
        qa)
            job_name="qa-e2e-tests"
            ;;
        e2e)
            job_name="nongroovy-e2e-tests-"
            ;;
        *)
            die "flavor $FLAVOR not supported"
            ;;
    esac

    case "$DATABASE" in
        postgres)
            job_name="postgres-$job_name"
            ;;
        rocksdb)
            ;;
        *)
            die "database $DATABASE not supported"
            ;;
    esac

    export CI_JOB_NAME="$job_name"
}

main "$@"
