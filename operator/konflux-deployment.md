# Something

0. Make sure you have [cert-manager installed](https://cert-manager.io/docs/installation/).
   It takes care of the TLS aspects of the connection from k8s API server to the webhook server
   embedded in the manager binary.

1. `ROX_PRODUCT_BRANDING=RHACS_BRANDING make deploy > resources.yaml`

2. Set `spec.template.spec.containers[0].image` to `quay.io/rhacs-eng/stackrox-operator:4.4.x-811-geb354f05a3-fast` (remove `-dirty`, replace 0 with x)

3. `cat resources.yaml| kubectl apply -f -`

4. Add missing `quay.io/rhacs-eng` pull secret to Controller Manager service account.

```bash
kubectl -n stackrox-operator-system create secret docker-registry quay-io-rhacs-eng-pull-secrets \
  --docker-server=https://quay.io/v2/ \
  --docker-username=... \
  --docker-password=...

kubectl -n stackrox-operator-system patch serviceaccount rhacs-operator-controller-manager -p '{"imagePullSecrets": [{"name": "quay-io-rhacs-eng-pull-secrets"}]}'
```

5. Create pull secret for stackrox deployments: `NAMESPACE="stackrox" make -C operator stackrox-image-pull-secret`

6. Apply custom resource for Central: `kubectl apply -n stackrox -f central-cr.yaml`

7. Generate init bundle on UI (or roxctl) and apply:

```bash
kubectl apply -f ~/Downloads/remote-Operator-secrets-cluster-init-bundle.yaml -n stackrox
```

8. Apply secured cluster resources:

```bash
kubectl apply -n stackrox -f secured-cluster-cr.yaml
```
