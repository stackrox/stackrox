# OpenShift Deployment Guide

This document provides instructions for deploying the ML Risk Service on OpenShift clusters.

## Security Context Constraints (SCC)

OpenShift uses Security Context Constraints to control pod security policies. The ML Risk Service has been configured to work with OpenShift's default SCCs.

### Default Deployment (Recommended)

The deployment uses dynamic UID assignment, which should work with the `restricted-v2` or `nonroot-v2` SCCs:

```bash
make k8s-deploy
```

The pod security context has been configured to:
- Use `runAsNonRoot: true` (no hardcoded UIDs)
- Allow OpenShift to assign UIDs dynamically
- Maintain security with read-only root filesystem
- Drop all capabilities

### Custom SCC (If Needed)

If the default deployment doesn't work due to SCC restrictions, you can apply a custom SCC:

1. **Apply the custom SCC** (requires cluster admin privileges):
   ```bash
   oc apply -f k8s/scc.yaml
   ```

2. **Update kustomization.yaml** to include the SCC:
   ```bash
   # Uncomment the scc.yaml line in kustomization.yaml
   sed -i 's/# - scc.yaml/- scc.yaml/' k8s/kustomization.yaml
   ```

3. **Deploy with custom SCC**:
   ```bash
   make k8s-deploy
   ```

## Troubleshooting

### Pod Scheduling Issues

If pods fail to schedule with SCC errors:

1. **Check pod events**:
   ```bash
   oc describe pod -l app.kubernetes.io/name=ml-risk-service -n stackrox
   ```

2. **Check which SCC is being used**:
   ```bash
   oc get pod -o jsonpath='{.metadata.annotations.openshift\.io/scc}' -l app.kubernetes.io/name=ml-risk-service -n stackrox
   ```

3. **View available SCCs**:
   ```bash
   oc get scc
   ```

### Permission Issues

If the service account lacks permissions:

1. **Check service account**:
   ```bash
   oc get sa ml-risk-service -n stackrox
   ```

2. **Check role bindings**:
   ```bash
   oc describe clusterrolebinding ml-risk-service
   oc describe rolebinding ml-risk-training -n stackrox
   ```

## Security Considerations

- The deployment runs as non-root with a read-only root filesystem
- All Linux capabilities are dropped for enhanced security
- Network policies restrict communication to essential services only
- Service account has minimal required permissions

## OpenShift-Specific Features

The deployment includes OpenShift-specific configurations:
- Compatible with OpenShift's security model
- Uses standard OpenShift SCCs where possible
- Includes proper labeling for OpenShift UI integration
- Configured for OpenShift's dynamic UID ranges

## Monitoring

The service exposes Prometheus metrics on port 8081:
- Metrics endpoint: `/metrics`
- Health endpoint: `/health`
- Readiness endpoint: `/ready`

To view in OpenShift console, the service is properly labeled for discovery.