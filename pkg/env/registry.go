package env

var (
	// DisableRegistryRepoList disables building and matching registry integrations using repo lists.
	DisableRegistryRepoList = RegisterBooleanSetting("ROX_DISABLE_REGISTRY_REPO_LIST", false)
)
