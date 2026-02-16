# Roxctl Authentication Reference

## Authentication Priority

When multiple authentication methods are configured, roxctl uses this priority:

1. **Command-line flags** (`--token-file`, `--password`)
2. **Environment variables** (`ROX_API_TOKEN`, `ROX_API_TOKEN_FILE`, `ROX_ADMIN_PASSWORD`)
3. **Local config** (from `roxctl central login`)

## API Token Authentication

### Creating API Tokens

API tokens are created in Central UI under **Platform Configuration > Integrations > Authentication Tokens**.

Token types:
* **Admin** - Full administrative access
* **Analyst** - Read-only access
* **Continuous Integration** - CI/CD operations
* **Sensor Creator** - Cluster onboarding
* **Custom** - Role-based permissions

### Using API Tokens

```bash
# Direct token value (less secure, visible in process list)
export ROX_API_TOKEN="eyJhbGciOiJSUzI1NiIs..."
roxctl central whoami

# Token file (recommended)
echo "eyJhbGciOiJSUzI1NiIs..." > ~/.rox/api-token
chmod 600 ~/.rox/api-token
export ROX_API_TOKEN_FILE=~/.rox/api-token
roxctl central whoami

# Command-line flag
roxctl --token-file ~/.rox/api-token central whoami
```

## Basic Authentication

Basic auth uses the `admin` user with the password set during Central installation.

```bash
# Environment variable
export ROX_ADMIN_PASSWORD="letmein"
roxctl central whoami

# Command-line flag
roxctl -p letmein central whoami
```

**Note:** Basic auth is disabled by default in production deployments. Use API tokens instead.

## Local Config (Interactive Login)

The `roxctl central login` command provides interactive authentication:

```bash
# Interactive login
roxctl central login

# Login to specific endpoint
roxctl central login -e central.example.com:443
```

This stores access and refresh tokens locally in `~/.roxctl/login.yaml`. The CLI automatically refreshes expired access tokens using the stored refresh token.

### Token Refresh

Tokens are automatically refreshed when:
* Access token has expired
* Refresh token is still valid

If the refresh token expires, you must run `roxctl central login` again.

## Machine-to-Machine (M2M) Token Exchange

For service accounts using OIDC tokens:

```bash
# Exchange OIDC token for Central access token
roxctl central m2m exchange --token "$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
```

This is useful for:
* Kubernetes service accounts
* CI/CD systems with OIDC providers
* Federated identity scenarios

## Connection Security

### TLS Options

```bash
# Skip TLS verification (development only)
roxctl --insecure-skip-tls-verify central whoami

# Use custom CA certificate
roxctl --ca /path/to/ca.pem central whoami

# Insecure mode (allows additional insecure options)
roxctl --insecure central whoami

# Plaintext (unencrypted, requires --insecure)
roxctl --insecure --plaintext central whoami
```

### Kubernetes Port-Forwarding

For clusters where Central is not externally exposed:

```bash
# Use current kubeconfig context
roxctl --use-current-k8s-context central whoami
```

This automatically sets up port-forwarding to Central.

## Best Practices

1. **CI/CD Pipelines**: Use API tokens with minimal required permissions
2. **Local Development**: Use `roxctl central login` for convenience
3. **Production Scripts**: Store tokens in secret managers, not environment variables
4. **Token Rotation**: Regularly rotate API tokens
5. **Audit**: Monitor token usage via Central audit logs

## Troubleshooting

### "unauthenticated" Error

1. Verify token/password is set: `echo $ROX_API_TOKEN`
2. Check token validity in Central UI
3. Ensure token has required permissions
4. Try explicit authentication: `roxctl -p <password> central whoami`

### "token expired" Error

1. Re-login: `roxctl central login`
2. Or use a new API token

### "permission denied" Error

1. Check token permissions in Central UI
2. Request appropriate role assignment
3. Use a different token with required permissions
