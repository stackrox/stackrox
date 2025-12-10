# Deploy ACS via operator-index image with Konflux images

```bash
ROX_PRODUCT_BRANDING=RHACS_BRANDING \
VERSION=4.10.0-nightly-20251210-fast \
IMG=quay.io/rhacs-eng/release-operator:4.10.0-nightly-20251210-fast \
INDEX_IMG_BASE=quay.io/rhacs-eng/stackrox-operator-index \
BUNDLE_IMG=quay.io/rhacs-eng/release-operator-bundle:v4.10.0-nightly-20251210-fast \
make index-build

docker push quay.io/rhacs-eng/stackrox-operator-index:v4.10.0-nightly-20251210-fast

export OPERATOR_CHANNEL="latest"
export INDEX_IMAGE="quay.io/rhacs-eng/stackrox-operator-index:v4.10.0-nightly-20251210-fast"
export MAIN_IMAGE_TAG="4.10.0-nightly-20251210-fast"
export MY_SECURED_CLUSTER_NAME="secured-cluster-1"
```

Then follow the instructions in the link below to deploy ACS with Konflux images.

https://spaces.redhat.com/spaces/StackRox/pages/483005167/How+to+deploy+ACS+with+Konflux+images#HowtodeployACSwithKonfluximages-OperatorIndex

## Secured cluster fails to deploy with the following status

```yaml
status:
  conditions:
    - lastTransitionTime: '2025-12-10T15:27:21Z'
      message: No deployments found
      reason: NoDeployments
      status: 'False'
      type: Available
    - lastTransitionTime: '2025-12-10T15:27:34Z'
      status: 'False'
      type: Deployed
    - lastTransitionTime: '2025-12-10T15:27:34Z'
      status: 'True'
      type: Initialized
    - lastTransitionTime: '2025-12-10T15:34:09Z'
      message: "failed pre-install: warning: Hook pre-install stackrox-secured-cluster-services/templates/stackrox-helm-configmap.yaml failed: 1 error occurred:\n\t* ConfigMap \"stackrox-secured-cluster-helm\" is invalid: metadata.labels: Invalid value: \"stackrox-secured-cluster-services-400.10.0-nightly-20251210-fast\": must be no more than 63 characters\n\n"
      reason: ReconcileError
      status: 'True'
      type: Irreconcilable
    - lastTransitionTime: '2025-12-10T15:27:34Z'
      status: 'False'
      type: Paused
    - lastTransitionTime: '2025-12-10T15:27:21Z'
      message: Spec changes pending reconciliation
      reason: Reconciling
      status: 'True'
      type: Progressing
    - lastTransitionTime: '2025-12-10T15:27:34Z'
      message: Proxy configuration has been applied successfully
      reason: ProxyConfigApplied
      status: 'False'
      type: ProxyConfigFailed
    - lastTransitionTime: '2025-12-10T15:27:34Z'
      message: "failed pre-install: warning: Hook pre-install stackrox-secured-cluster-services/templates/stackrox-helm-configmap.yaml failed: 1 error occurred:\n\t* ConfigMap \"stackrox-secured-cluster-helm\" is invalid: metadata.labels: Invalid value: \"stackrox-secured-cluster-services-400.10.0-nightly-20251210-fast\": must be no more than 63 characters\n\n"
      reason: InstallError
      status: 'True'
      type: ReleaseFailed
```
