# Roxctl Troubleshooting Reference

## Connection Issues

### Connection Refused

**Symptom:**
```
Error: could not connect to Central: dial tcp: connect: connection refused
```

**Causes & Solutions:**

1. **Central not running**
   ```bash
   # Check Central pod status
   kubectl get pods -n stackrox -l app=central

   # Check Central logs
   kubectl logs -n stackrox deploy/central --tail=50
   ```

2. **Wrong endpoint**
   ```bash
   # Verify endpoint
   echo $ROX_ENDPOINT

   # Test connectivity
   curl -k https://central.stackrox.svc:443/v1/ping
   ```

3. **Network policy blocking**
   ```bash
   # Check network policies
   kubectl get networkpolicies -n stackrox
   ```

### Certificate Errors

**Symptom:**
```
Error: x509: certificate signed by unknown authority
```

**Solutions:**

1. **Skip TLS verification (development only)**
   ```bash
   roxctl --insecure-skip-tls-verify central whoami
   # Or
   export ROX_INSECURE_CLIENT_SKIP_TLS_VERIFY=true
   ```

2. **Use custom CA certificate**
   ```bash
   # Extract CA from cluster
   kubectl get secret -n stackrox central-tls -o jsonpath='{.data.ca\.pem}' | base64 -d > ca.pem

   # Use with roxctl
   roxctl --ca ca.pem central whoami
   ```

3. **Wrong server name**
   ```bash
   # Specify correct SNI
   roxctl -e central-lb.example.com:443 -s central.stackrox.svc central whoami
   ```

### Timeout Errors

**Symptom:**
```
Error: context deadline exceeded
```

**Solutions:**

1. **Increase timeout**
   ```bash
   roxctl --timeout 5m image scan --image nginx:latest
   ```

2. **Network latency**
   ```bash
   # Use port-forwarding for direct connection
   roxctl --use-current-k8s-context central whoami
   ```

3. **Scan still in progress**
   ```bash
   # First scan of an image takes longer
   # Wait and retry, or use --force to ensure fresh scan
   roxctl image scan --image nginx:latest --force --timeout 10m
   ```

## Authentication Issues

### Unauthenticated Error

**Symptom:**
```
Error: rpc error: code = Unauthenticated desc = credentials not provided
```

**Solutions:**

1. **Check credentials are set**
   ```bash
   echo "Token: $ROX_API_TOKEN"
   echo "Token File: $ROX_API_TOKEN_FILE"
   echo "Password: $ROX_ADMIN_PASSWORD"
   ```

2. **Verify token is valid**
   ```bash
   # Check token in Central UI under Platform Configuration > Integrations
   # Or try logging in fresh
   roxctl central login
   ```

3. **Use explicit authentication**
   ```bash
   roxctl -p letmein central whoami
   # or
   roxctl --token-file /path/to/token central whoami
   ```

### Token Expired

**Symptom:**
```
Error: token has expired
```

**Solutions:**

1. **Re-login**
   ```bash
   roxctl central login
   ```

2. **Generate new API token** in Central UI

### Permission Denied

**Symptom:**
```
Error: rpc error: code = PermissionDenied desc = access denied
```

**Solutions:**

1. **Check token permissions** in Central UI
2. **Use token with appropriate role**
3. **Verify resource exists** (cluster, image, etc.)

## Image Scanning Issues

### Image Not Found

**Symptom:**
```
Error: could not pull image: unauthorized
```

**Solutions:**

1. **Provide registry credentials**
   ```bash
   roxctl image scan --image private.registry.io/app:v1 \
     --registry-username user \
     --registry-password pass
   ```

2. **Configure image pull secrets** in Central

### Scan Timeout

**Symptom:**
```
Error: timed out waiting for scan results
```

**Solutions:**

1. **Increase timeout**
   ```bash
   roxctl --timeout 15m image scan --image large-image:latest
   ```

2. **Check scanner health**
   ```bash
   kubectl get pods -n stackrox -l app=scanner
   kubectl logs -n stackrox deploy/scanner --tail=50
   ```

### No Vulnerabilities Found

**Possible causes:**

1. **Image already cached** - Use `--force` to rescan
2. **Scanner DB outdated** - Check scanner-db update status
3. **Unsupported package manager** - Some package types not detected

## Sensor/Cluster Issues

### Init Bundle Generation Fails

**Symptom:**
```
Error: failed to generate init bundle
```

**Solutions:**

1. **Check Central connectivity**
   ```bash
   roxctl central whoami
   ```

2. **Verify permissions** - Need `Cluster` write access

3. **Check for existing bundle** with same name
   ```bash
   roxctl central init-bundles list
   ```

### Sensor Not Connecting

**Symptom:** Cluster shows as "unhealthy" in Central

**Solutions:**

1. **Check sensor pod status**
   ```bash
   kubectl get pods -n stackrox -l app=sensor
   kubectl logs -n stackrox deploy/sensor --tail=100
   ```

2. **Verify init bundle was applied**
   ```bash
   kubectl get secrets -n stackrox | grep sensor
   ```

3. **Check Central endpoint** in sensor config
   ```bash
   kubectl get deploy sensor -n stackrox -o yaml | grep -A5 ROX_CENTRAL_ENDPOINT
   ```

4. **Regenerate certificates**
   ```bash
   roxctl sensor generate-certs <cluster-name>
   kubectl apply -f sensor-<cluster-name>/sensor-tls-secrets.yaml -n stackrox
   kubectl rollout restart deploy/sensor -n stackrox
   ```

## Common Errors Reference

| Error | Likely Cause | Quick Fix |
|-------|--------------|-----------|
| `connection refused` | Central not running | Check pod status |
| `x509: certificate signed by unknown authority` | TLS verification | Use `--insecure-skip-tls-verify` |
| `credentials not provided` | Missing auth | Set `ROX_API_TOKEN` or `ROX_ADMIN_PASSWORD` |
| `token has expired` | Expired token | Run `roxctl central login` |
| `permission denied` | Insufficient permissions | Check token role |
| `context deadline exceeded` | Timeout | Increase `--timeout` |
| `unauthorized` | Registry auth | Provide `--registry-username/password` |
| `image not found` | Wrong image reference | Verify image exists |
| `cluster not found` | Wrong cluster name | List clusters with API |

## Debug Commands

```bash
# Verbose output
roxctl --verbose central whoami

# Get Central version
roxctl version

# Test connectivity with minimal auth
curl -k https://central.stackrox.svc:443/v1/ping

# Download diagnostic bundle
roxctl central debug download-diagnostics

# Check API directly
curl -k -H "Authorization: Bearer $ROX_API_TOKEN" \
  https://central.stackrox.svc:443/v1/clusters
```

## Getting Help

```bash
# Command-specific help
roxctl central --help
roxctl image scan --help

# Generate shell completion
roxctl completion bash >> ~/.bashrc
```
