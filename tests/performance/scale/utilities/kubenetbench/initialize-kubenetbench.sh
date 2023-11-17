#!/bin/bash
set -e

# This script deploys kubenetbench. It does not actually generate load.
# To generate load run this script first and then generate-network-load.sh

artifacts_dir="$1"
test_name="$2"
knb_base_dir="$3"

DIR="$(cd "$(dirname "$0")" && pwd)"

source "$DIR"/../common.sh

[[ -n "$artifacts_dir" && -n "$test_name" ]] \
    || die "Usage: $0 <artifacts-dir> <test-name>"

if [[ -z "$knb_base_dir" ]]; then
    knb_base_dir="$(mktemp -d)"
fi

if [[ ! -d "$knb_base_dir" ]]; then
    mkdir -p "$knb_base_dir"
fi

knb_base_url="https://github.com/cilium/kubenetbench"
knb_url="${knb_base_url}/archive/refs/heads/master.zip"
knb_zip="$(mktemp)"
knb_dir="${knb_base_dir}/kubenetbench-master/"
wget "${knb_url}" -O "${knb_zip}"

unzip -d "${knb_base_dir}" "${knb_zip}"
rm -rf "${knb_zip}"

pushd "${knb_dir}"

# Patch tolerations in kubenetbench/core/monitor.go to not run on master nodes
patch -p1 << 'EOF'
diff --git a/kubenetbench/core/monitor.go b/kubenetbench/core/monitor.go
index edbe57b..0e914b2 100644
--- a/kubenetbench/core/monitor.go
+++ b/kubenetbench/core/monitor.go
@@ -39,11 +39,11 @@ spec:
         {{.sessLabel}}
         role: monitor
     spec:
-      # tolerations:
-      # # this toleration is to have the daemonset runnable on master nodes
-      # # remove it if your masters can't run pods
-      # - key: node-role.kubernetes.io/master
-      #   effect: NoSchedule
+      tolerations:
+      # this toleration is to have the daemonset runnable on master nodes
+      # remove it if your masters can't run pods
+      - key: node-role.kubernetes.io/master
+        effect: NoSchedule
EOF

log "build kubenetbench"
make

log "deploy kubenetbench"

export KUBECONFIG="${artifacts_dir}/kubeconfig"
kubenetbench/kubenetbench -s "${test_name}" init --port-forward
