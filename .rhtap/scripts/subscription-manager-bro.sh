#!/usr/bin/env bash

# This script is for registering container with Red Hat subscription-manager during RHTAP builds for getting access to
# RHEL RPMs during the build.
# The script was created as a workaround in absence of better options.
# TODO(ROX-20651): remove this script and switch to use content sets once available.

set -euo pipefail

SCRIPT_NAME="$(basename -- "${BASH_SOURCE[0]}")"
REPO_ROOT="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )"/../.. &> /dev/null && pwd )"

RHTAP_SECRET_NAME="subscription-manager-activation-key"
SECRET_KEY="activation-key"
SECRET_PATH="${REPO_ROOT}/.rhtap/build/${SECRET_KEY}"

RED_HAT_ORG_ID="11009103"
TARGET_DIR="/mnt"

# These were figured experimentally with the help of self-test subcommand.
TARGET_BACKUP_PATHS=(etc/pki/product-default etc/yum.repos.d etc/pki/entitlement var/lib/rhsm etc/pki/product etc/pki/consumer)


function main {
    if [[ "$#" == "0" ]] ; then
        >&2 echo "Error: command is missing. See usage below."
        usage
        exit 2
    fi

    local cmd="$1"
    shift

    local fn

    case "$cmd" in
    "help" | "--help" | "-h")
        fn=usage ;;
    "smuggle")
        fn=smuggle ;;
    "register")
        fn=register ;;
    "cleanup")
        fn=cleanup ;;
    "self-test")
        fn=self_test ;;
    "diff")
        fn=assert_diff ;;
    *)
        >&2 echo "Error: unknown command '$1'; call '$SCRIPT_NAME help' to see usage."
        exit 3
        ;;
    esac

    if [[ "$#" -gt "1" && "$cmd" != "diff" ]]; then
        >&2 echo "Error: too many arguments; call '$SCRIPT_NAME help' to see usage."
        exit 4
    fi

    $fn "$@"
}

function usage {
    echo "Usage: $SCRIPT_NAME smuggle|register|cleanup|self-test"
    echo
    echo "This script enables access to RHEL RPMs during RHTAP builds. The intended usage is as follows."
    echo

    echo -n "1. Make sure there is '$RHTAP_SECRET_NAME' secret in RHTAP with key name '$SECRET_KEY' and "
    echo "the actual activation key as a value."

    echo -n "2. In a Tekton pipeline step before the container build, copy the subscription manager activation "
    echo "key secret to the source workspace. Use:"
    echo "   \$ <source-workspace>/$SCRIPT_NAME smuggle"
    echo -n "   This expects '$RHTAP_SECRET_NAME' secret to be mounted as a workspace with the same name "
    echo "('$RHTAP_SECRET_NAME')."

    echo "3. Arrange Dockerfile stages so that there's UBI (normal) as installer and other RHEL/UBI (any) as target."
    echo "   Copy target contents to ${TARGET_DIR} in the installer stage. See self-test Dockerfiles as examples."

    echo "4. In installer stage, register the container with subscription manager. Use:"
    echo "   \$ $SCRIPT_NAME register"

    echo -n "5. Use 'dnf --installroot=${TARGET_DIR} ...' to install RHEL RPMs, enable RHEL modules, etc. "
    echo "in target contents."

    echo -n "6. In the same installer stage, deregister the container so that end users can't use "
    echo "our subscription on our behalf. Use:"
    echo "   \$ $SCRIPT_NAME cleanup"
    echo "   This step is mandatory because it cleans entitlements on target in the right way."

    echo -n "7. Copy out ${TARGET_DIR} contents from the installer stage to a new 'scratch' stage. "
    echo "That's your target container."

    echo
    echo -n "If you need to install entitled RPMs into multiple stages in one Dockerfile, use separate installer stage for each. "
    echo "This is because the script assumes one distinct system to be in ${TARGET_DIR}."
    # I did not bother to make it more universal since it's already a lot for a throw-away.

    echo
    echo "When altering this script, use self-test command as (regression) test tool:"
    echo "   \$ $SCRIPT_NAME self-test"
    echo "For it to work, you need to put a valid activation key in ${SECRET_PATH} file."
}

function smuggle {
    mkdir -p "${REPO_ROOT}/.rhtap/build/"
    cp --verbose "/workspace/${RHTAP_SECRET_NAME}/${SECRET_KEY}" "${SECRET_PATH}"
}

function register {
    if [[ ! -d "${TARGET_DIR}"/etc ]]; then
        >&2 echo "Error: Looks like target system is not mounted at ${TARGET_DIR}/etc"
        exit 5
    fi

    # Besides just installing packages and making the desired updates to rpmdb, the use of subscription-manager with
    # the subsequent installation introduces some side-effects that seem undesired. Backup and restore is how I suggest
    # maintaining the original state.
    echo "Backing up original artifacts in $TARGET_DIR"
    mkdir -p "${TARGET_DIR}"/tmp/restore
    tar --create -vf "${TARGET_DIR}"/tmp/restore/backup.tar --files-from /dev/null
    for item in "${TARGET_BACKUP_PATHS[@]}"; do
        if [[ -e "${TARGET_DIR}/${item}" ]]; then
            tar --append -vf "${TARGET_DIR}"/tmp/restore/backup.tar -C "${TARGET_DIR}" "${item}"
        fi
    done

    echo "Registering installer container with subscription manager"
    subscription-manager register --org="$RED_HAT_ORG_ID" --activationkey="$(cat "${SECRET_PATH}")"

    # It is suggested in the following articles that certain files can be linked to $TARGET_DIR/run/secrets,
    # but I was not able to make it work, therefore doing it differently.
    # https://www.neteye-blog.com/2022/07/how-to-use-a-hosts-redhat-subscription-to-run-containers-using-docker-instead-of-podman/
    # https://access.redhat.com/solutions/5870841
    echo "Enabling entitled rpm repos in $TARGET_DIR"
    mkdir -p "${TARGET_DIR}"/etc/pki/entitlement
    ln --verbose -s /etc/pki/entitlement/*.pem "${TARGET_DIR}"/etc/pki/entitlement
    ln --verbose --force -s /etc/yum.repos.d/redhat.repo "${TARGET_DIR}"/etc/yum.repos.d/

    echo "Looks like the registration succeeded. Don't forget to call '$SCRIPT_NAME cleanup' when done with rpms!"
}

function cleanup {
    echo "Cleaning up entitlement artifacts in $TARGET_DIR"

    echo "Restoring original artifacts"
    for item in "${TARGET_BACKUP_PATHS[@]}"; do
        rm --verbose -rf "${TARGET_DIR:?}/${item}"
    done
    tar --extract -vf "${TARGET_DIR}"/tmp/restore/backup.tar -C "${TARGET_DIR}"

    echo "Removing original artifacts backups"
    rm --verbose -rf "${TARGET_DIR:?}"/tmp/restore

    # It may be good to unregister this host container so that it's not left hanging in some Red Hat database.
    echo "Unregistering installer container"
    subscription-manager unregister

    echo "Cleanup complete."
}

function self_test {
    docker build -f "${REPO_ROOT}/.rhtap/scripts/bro.self-test-1.Dockerfile" "$REPO_ROOT"
    docker build -f "${REPO_ROOT}/.rhtap/scripts/bro.self-test-2.Dockerfile" "$REPO_ROOT"
}

function assert_diff {
    if [[ "$#" != "2" ]]; then
        >&2 echo "Error: expecting two arguments: expected and actual paths"
        exit 6
    fi

    local expected="$1"
    local actual="$2"

    local failed_check_file
    failed_check_file="$(mktemp)"

    echo "Comparing /etc"
    if ! diff --brief --recursive --no-dereference --exclude='ld.so.cache' "$expected"/etc "$actual"/etc ; then
        echo 1 >> "$failed_check_file"
    fi

    echo "Comparing /var"
    local var_exclusions
    var_exclusions="$(mktemp)"
    {
        # Before adding any exclusions here, make sure you check there's nothing sensitive in these files.
        # If sensitive, they should be added to backup/restore or cleanup.
        echo '/var/cache: (dnf|ldconfig)'
        echo '/var/lib: dnf'
        echo '/var/lib/dnf/history\.sqlite'
        echo '/var/lib/rpm(:|/)'
        echo '/var/log: (dnf.*|hawkey)\.log'
        echo '/var/log/hawkey\.log'
    } >> "$var_exclusions"

    if { diff --brief --recursive --no-dereference "$expected"/var "$actual"/var || true; } | \
        grep -vEf "$var_exclusions" | { grep '.'; }; then
        echo 2 >> "$failed_check_file"
    fi

    local other_dirs_to_compare=(bin home lib lib64 media mnt opt root sbin srv tmp usr)

    for dir in "${other_dirs_to_compare[@]}"; do
        echo "Comparing /$dir"
        if ! diff --brief --recursive --no-dereference "$expected/$dir" "$actual/$dir"; then
            echo 3 >> "$failed_check_file"
        fi
    done

    if [[ -s "$failed_check_file" ]]; then
        >&2 echo "Error: differences detected"
        exit 7
    fi

    echo "Diff check for $expected and $actual passed"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
