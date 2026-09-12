---
name: test-scanner-removal-helm
description: "Reference for deploying ACS via Helm for Scanner V2 removal testing: chart rendering, install, upgrade, CRS, cleanup"
disable-model-invocation: true
---

# ACS Helm Deployment Reference

How to deploy and manage ACS via Helm for testing Scanner V2 removal.

## Charts

### 5.0 charts (current branch)

Render from local meta-templates using roxctl. Uses `quay.io/stackrox-io` registry.

```bash
roxctl helm output central-services \
  --image-defaults=opensource --debug \
  --output-dir=/tmp/stackrox-central-5.0-chart --remove

roxctl helm output secured-cluster-services \
  --image-defaults=opensource --debug \
  --output-dir=/tmp/stackrox-secured-cluster-5.0-chart --remove
```

If roxctl is outdated, rebuild: `make cli_host-arch`

### 4.11.3 charts (pre-upgrade baseline)

Pull from the `stackrox` Helm repo. Same `quay.io/stackrox-io` registry, tag `4.11.3`.

```bash
helm repo list | grep stackrox || \
  helm repo add stackrox https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/
helm repo update stackrox

helm pull stackrox/stackrox-central-services \
  --version 400.11.3 --untar --untardir /tmp/helm-4.11.3

helm pull stackrox/stackrox-secured-cluster-services \
  --version 400.11.3 --untar --untardir /tmp/helm-4.11.3
```

Chart paths: `/tmp/helm-4.11.3/stackrox-central-services`, `/tmp/helm-4.11.3/stackrox-secured-cluster-services`.

## Scanner Helm values

| Setting | V2 Flag | V4 Flag |
|---|---|---|
| default | (omit) | (omit) |
| enabled | `--set scanner.disable=false` | `--set scannerV4.disable=false` |
| disabled | `--set scanner.disable=true` | `--set scannerV4.disable=true` |

In 5.0 charts, the V2 flag is ignored (Scanner V2 is removed) but can be set to test graceful handling.

## Install Central Services

```bash
helm install stackrox-central-services <chart-path> \
  --namespace stackrox --create-namespace \
  --set central.adminPassword.value="letmein123" \
  [scanner flags]
```

## Generate CRS (Cluster Registration Secret)

Required before installing a Secured Cluster. Central must be running.

```bash
kubectl rollout status deployment/central -n stackrox --timeout=300s

kubectl -n stackrox port-forward svc/central 18443:443 &
PF_PID=$!
sleep 2

roxctl -e https://localhost:18443 -p 'letmein123' --insecure-skip-tls-verify \
  central crs generate test-cluster \
  --output /tmp/test-cluster-crs.yaml

kill $PF_PID
```

The admin password can also be extracted from the Helm release secret:
```bash
kubectl -n stackrox get secret sh.helm.release.v1.stackrox-central-services.v1 \
  -o jsonpath='{.data.release}' | base64 --decode | base64 --decode | gunzip | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['config']['central']['adminPassword']['value'])"
```

## Install Secured Cluster

### Co-located (same namespace as Central)

The chart auto-detects Central and skips deploying scanners from the secured-cluster
chart. Sensor is configured to use Central's scanner instances.

```bash
helm install stackrox-secured-cluster-services <sc-chart-path> \
  --namespace stackrox \
  --set-file crs.file=/tmp/test-cluster-crs.yaml \
  --set clusterName=test-cluster \
  --set centralEndpoint=central.stackrox.svc:443 \
  [scanner flags]
```

### Standalone (different namespace)

Deploys its own scanner-v4-indexer (no matcher — matcher only runs on central).

```bash
helm install stackrox-secured-cluster-services <sc-chart-path> \
  --namespace stackrox-secured --create-namespace \
  --set-file crs.file=/tmp/test-cluster-crs.yaml \
  --set clusterName=test-cluster \
  --set centralEndpoint=central.stackrox.svc:443 \
  [scanner flags]
```

## Upgrade (4.11.3 → 5.0)

Install 4.11.3 first (see "Install Central Services" above with the 4.11.3 chart path).
Only wait for Central to be ready — no need to wait for Scanner V2 or V4 deployments.
We only need Central running to generate a CRS for secured cluster tests.

```bash
helm upgrade stackrox-central-services /tmp/stackrox-central-5.0-chart \
  --namespace stackrox \
  --reuse-values \
  --force-conflicts
```

- `--reuse-values` carries over existing settings (including old V2 scanner flags).
- `--force-conflicts` is required on OCP with Helm 4.x — OpenShift's
  image-registry-pull-secrets controller conflicts with server-side apply on
  ServiceAccount `.imagePullSecrets`.

Secured cluster upgrade (same pattern):
```bash
helm upgrade stackrox-secured-cluster-services /tmp/stackrox-secured-cluster-5.0-chart \
  --namespace <ns> \
  --reuse-values \
  --force-conflicts
```

## Verification commands

### Check deployments

```bash
kubectl get deployments -n <ns> \
  -o custom-columns='NAME:.metadata.name,READY:.status.readyReplicas,DESIRED:.spec.replicas,IMAGE:.spec.template.spec.containers[0].image'
```

Scanner deployment names: `scanner`, `scanner-db` (V2), `scanner-v4-indexer`,
`scanner-v4-matcher`, `scanner-v4-db` (V4).

### Check Central env vars

```bash
kubectl get deployment central -n <ns> \
  -o jsonpath='{range .spec.template.spec.containers[?(@.name=="central")].env[*]}{.name}={.value}{"\n"}{end}' | \
  grep -E "^ROX_(LEGACY_SCANNER|SCANNER_V4)="
```

Post-5.0: `ROX_LEGACY_SCANNER` should be `false`, `ROX_SCANNER_V4` should be `true`
(when V4 is enabled).

### Check sensor env vars

```bash
kubectl get deployment sensor -n <ns> \
  -o jsonpath='{range .spec.template.spec.containers[?(@.name=="sensor")].env[*]}{.name}={.value}{"\n"}{end}' | \
  grep -E "^ROX_(LOCAL_IMAGE_SCANNING_ENABLED|SCANNER_V4|SCANNER_GRPC_ENDPOINT)="
```

Key env vars: `ROX_LOCAL_IMAGE_SCANNING_ENABLED`, `ROX_SCANNER_V4`,
`ROX_SCANNER_V4_INDEXER_ENDPOINT`. Post-5.0, `ROX_SCANNER_GRPC_ENDPOINT` should
never be set.

## Cleanup

```bash
helm uninstall stackrox-secured-cluster-services -n stackrox-secured 2>/dev/null
helm uninstall stackrox-secured-cluster-services -n stackrox 2>/dev/null
helm uninstall stackrox-central-services -n stackrox
kubectl delete pvc --all -n stackrox 2>/dev/null
kubectl delete pvc --all -n stackrox-secured 2>/dev/null
kubectl delete ns stackrox stackrox-secured 2>/dev/null
```
