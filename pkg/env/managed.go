package env

var (
	// ManagedCentral is set to true to signal that the central is running as a managed instance.
	ManagedCentral = RegisterBooleanSetting("ROX_MANAGED_CENTRAL", false)

	ACSCSEmailURL = RegisterSetting("ROX_ACSCS_EMAIL_URL", WithDefault("https://emailsender.rhacs.svc.cluster.local"))

	// TenantID is set from the according central pod label with the Downward API.
	TenantID = RegisterSetting("ROX_TENANT_ID", AllowEmpty())
)
