package env

import "time"

var (
	// DisableRegistryRepoList disables building and matching registry integrations using
	// repo lists (`/v2/_catalog`).
	DisableRegistryRepoList = RegisterBooleanSetting("ROX_DISABLE_REGISTRY_REPO_LIST", false)

	// RegistryDialerTimeout is the net.Dialer timeout of the client transport.
	// It limits the time the dialer attempts the connection. The timeout is
	// chosen as a small value to prevent unavailable registries from blocking
	// image scanning.
	RegistryDialerTimeout = registerDurationSetting("ROX_REGISTRY_DIALER_TIMEOUT", 5*time.Second)

	// RegistryResponseTimeout is the response header timeout of the client transport.
	// It limits the time to wait for a server's response headers after fully
	// writing the request.
	RegistryResponseTimeout = registerDurationSetting("ROX_REGISTRY_RESPONSE_TIMEOUT", 60*time.Second)

	// RegistryClientTimeout is used as http.Client.Timeout for the registry's HTTP
	// client and hence includes everything from connection to reading the
	// response body. The timeout has been chosen rather arbitrarily, it is
	// probably less harm in waiting a bit longer than in aborting early a
	// request that is about to succeed.
	RegistryClientTimeout = registerDurationSetting("ROX_REGISTRY_CLIENT_TIMEOUT", 90*time.Second)
)
