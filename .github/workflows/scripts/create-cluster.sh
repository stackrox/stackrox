#!/usr/bin/env bash
#
# Creates a cluster on infra.
#

set -euo pipefail

FLAVOR="$1"
NAME="$2"
LIFESPAN="$3"
WAIT="$4"

if [ "$#" -gt 4 ]; then
    ARGS="$5"
else
    ARGS=""
fi

check_not_empty \
    FLAVOR NAME LIFESPAN WAIT \
    INFRA_TOKEN

CNAME="${NAME//./-}"

function cluster_info() {
    infractl 2>/dev/null get "$1" --json
}

function cluster_status() {
    cluster_info "$1" | jq -r '.Status'
}

function cluster_destroying() {
    [ "$(cluster_status "$1")" -eq 3 ]
}

function infra_status_summary() {
    gh_summary <<EOF
*$2*
Infra status for '$1':
\`\`\`$(cluster_info "$1")\`\`\`

EOF
}

case $(cluster_status "$CNAME") in
1)
    # Don't wait for the cluster being created, as another workflow could be
    # waiting for it.
    # TODO: use concurrency tweak to allow only single workflow running at once.
    infra_status_summary "$CNAME" "Cluster is being created by another workflow"
    exit 0
    ;;
2)
    # Cluster exists already.
    infra_status_summary "$CNAME" "Cluster already exists"
    exit 0
    ;;
3)
    # Cluster is being destroyed.
    infra_status_summary "$CNAME" "Cluster is being destroyed"
    while cluster_destroying "$CNAME"; do
        gh_log notice "Waiting 30s for the cluster '$CNAME' to be destroyed"
        sleep 30
    done
    ;;
4)
    # Cluster has already been destroyed. Create it again.
    gh_log notice "Cluster \`$CNAME\` has been destroyed already."
    infra_status_summary "$CNAME" "Cluster has been destroyed already"
    ;;
*)
    infra_status_summary "$CNAME" "Unknown status"
    ;;
esac

# Creating a cluster
echo "Will attempt to create the cluster"

OPTIONS=()
if [ "$WAIT" = "true" ]; then
    OPTIONS+=("--wait")
    gh_log warning "The job will wait for the cluster creation to finish."
fi

IFS=',' read -ra args <<<"$ARGS"
for arg in "${args[@]}"; do
    OPTIONS+=("--arg")
    OPTIONS+=("$arg")
done

infractl create "$FLAVOR" "$CNAME" \
    --lifespan "$LIFESPAN" \
    "${OPTIONS[@]}"

infra_status_summary "$CNAME" "Cluster creation has been requested"
