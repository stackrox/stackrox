#!/usr/bin/env bash
set -eo pipefail

# Mount local binaries to enable HOTRELOAD after a deployment.
# It helps to run recently build binaries (i.e. from `make fast-central`) inside the cluster by
# only deleting the pod, instead of building a new main image.
# Usage: ./enable-hotreload.sh [sensor,central,migrator,admission]

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [[ -z "$1" ]]; then
  echo "Usage: $0 <component> [ <namespace> ]"
  echo "Available components: sensor, central, migrator, admission"
  exit 1
fi

# realpath binary might be missing on MacOS
function realpath {
	[[ -n "$1" ]] || return 0
	python3 -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "$1"
}

function hotload_binary {
  local binary_name="$1"
  local local_name="$2"
  local deployment="$3"
  local namespace=${4:-stackrox}

  echo
  echo "**********"
  echo "$binary_name binary is being hot reloaded"
  echo "**********"
  echo

  goarch=$(go env GOARCH)
  binary_path=$(realpath "$(git rev-parse --show-toplevel)/bin/linux_${goarch}/${local_name}")
  # set custom source path, e.g. when StackRox source is mounted into kind.
  if [[ -n "$ROX_LOCAL_SOURCE_PATH" ]]; then
      echo "ROX_LOCAL_SOURCE_PATH is set to $ROX_LOCAL_SOURCE_PATH."
      binary_path="$ROX_LOCAL_SOURCE_PATH/bin/linux_${goarch}/${local_name}"
  fi

  kubectl -n "${namespace}" patch "deploy/${deployment}" --patch-file <(cat <<EOF
spec:
  template:
    spec:
      containers:
        - name: "${deployment}"
          volumeMounts:
            - mountPath: "/stackrox/${binary_name}"
              name: "binary-${local_name}"
      volumes:
        - hostPath:
            path: "${binary_path}"
            type: ""
          name: "binary-${local_name}"
EOF
)
  kubectl -n "${namespace}" set env "deploy/${deployment}" "ROX_HOTRELOAD=true"
}

component="$1"
namespace=${2:-stackrox}

case "${component}" in
"sensor")
  hotload_binary bin/kubernetes-sensor kubernetes sensor "${namespace}"
  ;;
"central")
  hotload_binary central central central "${namespace}"
  ;;
"migrator")
  hotload_binary bin/migrator migrator central "${namespace}"
  ;;
"admission"|"admission-control"|"admission-controller")
  hotload_binary admission-control admission-control admission-control "${namespace}"
  ;;
*)
  echo "Invalid component: ${component}"
  echo "Available components: sensor, central, migrator, admission"
  exit 1
  ;;
esac
