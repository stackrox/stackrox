package env

import "time"

var (
	// DisableRegistryRepoList disables building and matching registry integrations using
	// repo lists (`/v2/_catalog`).
	DisableRegistryRepoList = RegisterBooleanSetting("ROX_DISABLE_REGISTRY_REPO_LIST", false)

	// RegistryDialerTimeout is the net.Dialer timeout of the client transport.
	// It limits the time the dialer attempts the connection. The timeout is
	// chosen as a small value to prevent unavailable registries from blocking
	// image scanning. When connecting to a proxy, this timeout only limits the
	// connection attempt to the proxy and not the proxy target.
	RegistryDialerTimeout = registerDurationSetting("ROX_REGISTRY_DIALER_TIMEOUT", 5*time.Second)

	// RegistryResponseTimeout is the response header timeout of the client transport.
	// It limits the time to wait for a server's response headers after fully
	// writing the request.
	RegistryResponseTimeout = registerDurationSetting("ROX_REGISTRY_RESPONSE_TIMEOUT", 10*time.Second)

	// RegistryClientTimeout is used as http.Client.Timeout for the registry's HTTP
	// client and hence includes everything from connection to reading the
	// response body. This timeout must not be chosen too large because it serves
	// as a connection timeout for proxied registries.
	RegistryClientTimeout = registerDurationSetting("ROX_REGISTRY_CLIENT_TIMEOUT", 10*time.Second)

	// DedupeImageIntegrations when enabled will lead to deduping registry integrations
	// in cases where the URL and other relevant configuration for accessing the registry such
	// as the username, password are duplicated across integrations. This will lead to less
	// registry calls being made during scanning by ACS.
	DedupeImageIntegrations = RegisterBooleanSetting("ROX_DEDUPE_IMAGE_INTEGRATIONS", true)

	// RegistryTLSCheckTTL will set the duration for which registry TLS checks will be cached.
	RegistryTLSCheckTTL = registerDurationSetting("ROX_REGISTRY_TLS_CHECK_CACHE_TTL", 15*time.Minute)

	// RegistryECRTokenRefreshTimeout bounds how long the ECR transport waits for
	// the AWS SDK GetAuthorizationToken call (STS/ECR token exchange) before
	// giving up. This call is made with a context that is not bounded by any
	// caller deadline (the vendored docker-registry-client never threads a real
	// request context through), so it must be bounded here independently.
	RegistryECRTokenRefreshTimeout = registerDurationSetting("ROX_REGISTRY_ECR_TOKEN_REFRESH_TIMEOUT", 10*time.Second)
)
