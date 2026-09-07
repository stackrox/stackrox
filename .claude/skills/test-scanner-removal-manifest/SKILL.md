---
name: test-scanner-removal-manifest
description: "Reference for deploying ACS via manifests (roxctl generate) for Scanner V2 removal testing: install, upgrade, cleanup"
disable-model-invocation: true
---

# ACS Manifest Deployment Reference

How to deploy and manage ACS via manifests for testing Scanner V2 removal.
Only Central Services topology is supported — Secured Cluster and Co-located
are not supported via manifests.

## Prerequisites

Two roxctl binaries are needed for upgrade testing:

- **4.11.3 roxctl**: built from the 4.11.3 worktree at `/tmp/stackrox-4.11.3/bin/darwin_arm64/roxctl`
- **5.0 roxctl**: built from the current branch at `bin/darwin_arm64/roxctl`

To build roxctl: `make cli_host-arch`

Important: do **not** use `--debug` when generating bundles — that reads templates
from the local filesystem instead of the embedded ones, which produces wrong results.

## Generate manifest bundle

```bash
# 4.11.3
/tmp/stackrox-4.11.3/bin/darwin_arm64/roxctl central generate openshift pvc \
  --image-defaults=opensource \
  --output-dir=/tmp/central-bundle-4.11.3 \
  -p "letmein123"

# 5.0
bin/darwin_arm64/roxctl central generate openshift pvc \
  --image-defaults=opensource \
  --output-dir=/tmp/central-bundle-5.0 \
  -p "letmein123"
```

Uses `quay.io/stackrox-io` registry. For k8s (non-OpenShift) clusters, replace
`openshift` with `k8s`.

### Bundle structure

The bundle contains directories: `central/`, `scanner/`, `scanner-v4/`, `helm/`.

- **4.11.3**: `scanner/` has YAML manifests (Scanner V2). `scanner-v4/` has YAML manifests.
- **5.0**: `scanner/` has only scripts, no YAML (V2 removed). `scanner-v4/` has YAML manifests.

## Install

```bash
cd /tmp/central-bundle-<version>

# 1. Setup namespace and pull secrets
./central/scripts/setup.sh

# 2. Deploy Central
oc create -R -f central

# 3. Deploy Scanner V2 (4.11.3 only — skip for 5.0)
./scanner/scripts/setup.sh
oc create -R -f scanner

# 4. Deploy Scanner V4
./scanner-v4/scripts/setup.sh
oc create -R -f scanner-v4
```

The `setup.sh` scripts may fail to create pull secrets on OCP clusters that use
the global pull secret — this is fine, pods will still pull images.

To skip Scanner V4 deployment, simply don't apply the `scanner-v4/` directory.

## Upgrade (4.11.3 → 5.0)

Apply the 5.0 bundle on top of the existing 4.11.3 deployment:

```bash
cd /tmp/central-bundle-5.0

./central/scripts/setup.sh
oc apply -R -f central
oc apply -R -f scanner-v4
```

**Scanner V2 is NOT removed by this upgrade.** Manifest-based upgrades only apply
what's in the new bundle — they don't delete orphaned resources. Scanner V2
deployments (`scanner`, `scanner-db`) remain running with old images. This is
expected (⚠️ in the test matrix) and requires manual removal:

```bash
oc delete deployment scanner scanner-db -n stackrox
oc delete service scanner scanner-db -n stackrox
oc delete hpa scanner -n stackrox
oc delete secret scanner-tls scanner-db-tls -n stackrox 2>/dev/null
oc delete configmap scanner-config -n stackrox 2>/dev/null
oc delete networkpolicy scanner scanner-db -n stackrox 2>/dev/null
```

## Verification commands

### Check deployments

```bash
kubectl get deployments -n stackrox \
  -o custom-columns='NAME:.metadata.name,READY:.status.readyReplicas,DESIRED:.spec.replicas,IMAGE:.spec.template.spec.containers[0].image'
```

Scanner deployment names: `scanner`, `scanner-db` (V2), `scanner-v4-indexer`,
`scanner-v4-matcher`, `scanner-v4-db` (V4).

### Check Central env vars

```bash
kubectl get deployment central -n stackrox \
  -o jsonpath='{range .spec.template.spec.containers[?(@.name=="central")].env[*]}{.name}={.value}{"\n"}{end}' | \
  grep -E "^ROX_(LEGACY_SCANNER|SCANNER_V4)="
```

Post-5.0: `ROX_LEGACY_SCANNER` should be `false`, `ROX_SCANNER_V4` should be `true`
(when V4 is enabled).

## Cleanup

```bash
oc delete ns stackrox
```
