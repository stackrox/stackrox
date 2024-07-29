#!/usr/bin/env bash

# This script is for registering container with Red Hat subscription-manager during Konflux builds for getting access
# to RHEL RPMs during the build.
# The script was created as a workaround in absence of better options.
# TODO(ROX-20651): remove this script and switch to use content sets once available.

set -euo pipefail

SCRIPT_NAME="$(basename -- "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

SECRET_NAME_IN_KONFLUX="subscription-manager-activation-key"
# The mount is provided by the buildah task when the ACTIVATION_KEY parameter is set to a valid secret name.
SECRET_MOUNT_PATH="/activation-key"
SECRET_KEY="activation-key"
SECRET_LOCAL_PATH="${SECRET_MOUNT_PATH}/${SECRET_KEY}"
SECRET_INFO_URL='https://docs.engineering.redhat.com/pages/viewpage.action?pageId=407312060'

RED_HAT_ORG_ID="11009103"
TARGETS_LIST_FILE="/tmp/subscription-manager-bro-targets"

# These were figured experimentally with the help of self-test subcommand.
TARGET_BACKUP_PATHS=(
    etc/pki/consumer
    etc/pki/entitlement
    etc/pki/product
    etc/pki/product-default
    etc/yum.repos.d
    var/lib/rhsm
    var/cache/ldconfig
)


function main {
    if [[ "$#" == "0" ]] ; then
        >&2 echo "Error: command is missing. See the usage below."
        usage
        exit 2
    fi

    local cmd="$1"
    shift

    local fn

    case "$cmd" in
    "help" | "--help" | "-h")
        fn=usage ;;
    "register")
        fn=register ;;
    "cleanup")
        fn=cleanup ;;
    "self-test")
        fn=self_test ;;
    "diff")
        fn=assert_diff ;;
    *)
        >&2 echo "Error: unknown command '$1'; call '$SCRIPT_NAME help' to see the usage."
        exit 2
        ;;
    esac

    if [[ "$#" -gt "0" && "$cmd" != "diff" && "$cmd" != "register" ]]; then
        >&2 echo "Error: too many arguments; call '$SCRIPT_NAME help' to see the usage."
        exit 2
    fi

    "$fn" "$@"
}

function usage {
    local example_target_dir="/mnt"

    echo "Usage: $SCRIPT_NAME register|cleanup|self-test"
    echo
    echo "This script enables access to RHEL RPMs during Konflux builds. The intended usage is as follows."
    echo

    echo -n "1. Make sure there is a '$SECRET_NAME_IN_KONFLUX' secret in Konflux with key name '$SECRET_KEY' and "
    echo "the actual activation key as a value."
    echo "   Find where to get the secret from ${SECRET_INFO_URL}"

    echo -n "2. In the Tekton pipeline 'build-container' step that uses the 'buildah' task, provide the "
    echo "'$SECRET_NAME_IN_KONFLUX' secret name for the 'ACTIVATION_KEY' parameter."

    echo "3. Arrange Dockerfile stages to have UBI (normal) as an installer and other RHEL/UBI (any) as a target."
    echo "   Make sure to match major versions: 8/8 is ok but 9/8 or 8/9 will result in errors."
    echo "   Copy the target contents to some directory, e.g. ${example_target_dir}, in the installer stage."
    echo "   See self-test Dockerfiles as examples."

    echo "4. In the installer stage, register the container with the subscription manager. Use:"
    echo "   \$ $SCRIPT_NAME register ${example_target_dir}"
    echo -n "   It is possible to provide multiple target directories as arguments if the script is used to prepare "
    echo "multiple distinct stages."

    echo -n "5. Use 'dnf --installroot=${example_target_dir} ...' to install RHEL RPMs, enable RHEL modules, etc. "
    echo "in the target contents."

    echo -n "6. In the same installer stage, deregister the container so that the end users can't use "
    echo "our subscription on our behalf. Use:"
    echo "   \$ $SCRIPT_NAME cleanup"
    echo "   This step is mandatory because it cleans entitlements on the target in the right way."

    echo -n "7. Copy out ${example_target_dir} contents from the installer stage to a new 'scratch' stage. "
    echo "That's your target container."

    echo
    echo "When altering this script, use the 'self-test' command as a (regression) test tool:"
    echo "   \$ $SCRIPT_NAME self-test"
    echo "For it to work, you need to put a valid activation key in ${SCRIPT_DIR}/${SECRET_KEY} file."
    echo "Find out where to get it from ${SECRET_INFO_URL}"
}

function register {
    if [[ ! -s "${SECRET_LOCAL_PATH}" ]]; then
        >&2 echo "Error: it does not look like the activation key is present in ${SECRET_LOCAL_PATH}"
        exit 3
    fi
    local secret
    secret="$(cat "${SECRET_LOCAL_PATH}")"

    if [[ "$#" -lt 1 ]]; then
        >&2 echo "Error: target path(s) must be provided for the 'register' command."
        exit 2
    fi

    local target_dirs=( "$@" )

    check_targets_and_store_paths_for_cleanup "${target_dirs[@]}"

    # Besides just installing packages and making the desired updates to rpmdb, the use of subscription-manager with
    # the subsequent installation introduces some side-effects that seem undesired. Backup and restore is how I suggest
    # maintaining the original state of the target.
    for target_dir in "${target_dirs[@]}"; do
        echo "Backing up the original artifacts in $target_dir"
        mkdir -p "${target_dir}/tmp/restore"
        tar --create -vf "${target_dir}/tmp/restore/backup.tar" --files-from /dev/null
        for item in "${TARGET_BACKUP_PATHS[@]}"; do
            if [[ -e "${target_dir}/${item}" ]]; then
                tar --append -vf "${target_dir}/tmp/restore/backup.tar" -C "${target_dir}" "${item}"
            fi
        done
    done

    echo "Registering the installer container with the subscription manager"
    subscription-manager register --org="$RED_HAT_ORG_ID" --activationkey="$secret"

    # It is suggested in the following articles that certain files can be linked to $target_dir/run/secrets,
    # but I was not able to make it work, therefore doing it differently.
    # https://www.neteye-blog.com/2022/07/how-to-use-a-hosts-redhat-subscription-to-run-containers-using-docker-instead-of-podman/
    # https://access.redhat.com/solutions/5870841
    for target_dir in "${target_dirs[@]}"; do
        echo "Enabling entitled rpm repos in $target_dir"
        mkdir -p "${target_dir}/etc/pki/entitlement"
        ln --verbose -s /etc/pki/entitlement/*.pem "${target_dir}/etc/pki/entitlement"
        ln --verbose --force -s /etc/yum.repos.d/redhat.repo "${target_dir}/etc/yum.repos.d/"
    done

    echo "Looks like the registration succeeded. Don't forget to call '$SCRIPT_NAME cleanup' when done with rpms!"
}

function check_targets_and_store_paths_for_cleanup {
    local target_dirs=( "$@" )

    for target_dir in "${target_dirs[@]}"; do
        if [[ ! -d "${target_dir}/etc" ]]; then
            >&2 echo "Error: Looks like the target system is not placed at ${target_dir}"
            exit 4
        fi
    done

    if [[ -f "${TARGETS_LIST_FILE}" ]]; then
        >&2 echo "Error: ${TARGETS_LIST_FILE} already exists. Are you trying to register again without doing a cleanup?"
        exit 5
    fi

    printf "%s\n" "${target_dirs[@]}" > "${TARGETS_LIST_FILE}"
}

function cleanup {
    local -a target_dirs
    readarray -t target_dirs < "${TARGETS_LIST_FILE}"

    for target_dir in "${target_dirs[@]}"; do
        echo "Cleaning up entitlement artifacts in $target_dir"

        echo "Restoring original artifacts"
        for item in "${TARGET_BACKUP_PATHS[@]}"; do
            rm --verbose -rf "${target_dir:?}/${item}"
        done
        tar --extract -vf "${target_dir}/tmp/restore/backup.tar" -C "${target_dir}"

        echo "Removing original artifacts backups"
        rm --verbose -rf "${target_dir:?}/tmp/restore"
    done

    # It should be good to unregister this installer container so that it's not left hanging in some Red Hat database.
    echo "Unregistering the installer container"
    subscription-manager unregister

    rm --verbose "${TARGETS_LIST_FILE}"

    echo "Cleanup complete."
}

function self_test {
    local command="podman"

    local targets=(
        "registry.access.redhat.com/ubi8/ubi-micro:latest"
        "registry.access.redhat.com/ubi8/ubi-minimal:latest"
        "registry.access.redhat.com/ubi8/ubi:latest"
        "registry.redhat.io/rhel8/toolbox:latest"

        "registry.access.redhat.com/ubi9/ubi-micro:latest"
        "registry.access.redhat.com/ubi9/ubi-minimal:latest"
        "registry.access.redhat.com/ubi9/ubi:latest"
        "registry.redhat.io/rhel9/toolbox:latest"
    )

    for target in "${targets[@]}"; do
        [[ $target =~ /(ubi|rhel)([0-9]+)/ ]]
        local major_version="${BASH_REMATCH[2]}"

        echo
        echo
        echo "Testing against ${target} with the installer major version ${major_version}"
        echo
        echo

        set -x
        "${command}" build \
            -f "${SCRIPT_DIR}/bro.self-test.Dockerfile" \
            --build-arg TARGET_BASE="${target}" \
            --build-arg INSTALLER_MAJOR_VERSION="${major_version}" \
            "${SCRIPT_DIR}"
        set +x
    done

    "${command}" build -f "${SCRIPT_DIR}/bro.self-test-demo.Dockerfile" "${SCRIPT_DIR}"
    "${command}" build -f "${SCRIPT_DIR}/bro.self-test-multiple-targets.Dockerfile" "${SCRIPT_DIR}"
    echo "Self-tests passed."
}

function assert_diff {
    if [[ "$#" != "2" ]]; then
        >&2 echo "Error: expecting two arguments: expected and actual paths"
        exit 2
    fi

    local expected="$1"
    local actual="$2"

    local failed_check_file
    failed_check_file="$(mktemp)"

    echo "Comparing /etc"
    if ! diff --brief --recursive --no-dereference --exclude='ld.so.cache' "$expected/etc" "$actual/etc" ; then
        echo 1 >> "$failed_check_file"
    fi

    echo "Comparing /var"
    local var_exclusions
    var_exclusions="$(mktemp)"
    {
        # Before adding any exclusions here, make sure you check there's nothing sensitive in these files.
        # If sensitive, they should be added to backup/restore (TARGET_BACKUP_PATHS) or cleanup.
        echo '/var/lib: dnf'
        echo '/var/lib/dnf/history\.sqlite'
        echo '/var/lib/rpm(:|/)'
        echo '/var/log: (dnf.*|hawkey)\.log'
        echo '/var/log/(dnf.*|hawkey)\.log'
        # /var/cache/dnf should be kept on Target, otherwise Konflux Enterprise Check fails not finding SBOM.
        echo '/var/cache(: |/)dnf'
    } >> "$var_exclusions"

    if { diff --brief --recursive --no-dereference "$expected/var" "$actual/var" || true; } | \
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
        exit 6
    fi

    echo "Diff check for $expected and $actual passed."
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
