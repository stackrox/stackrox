package env

var (
	// ManagedCentral is set to true to signal that the central is running as a managed instance.
	ManagedCentral = RegisterBooleanSetting("ROX_MANAGED_CENTRAL", false)

	// TenantID is set from the according central pod label with the Downward API.
	TenantID = RegisterSetting("ROX_TENANT_ID", AllowEmpty())

	// ExpiredAt is set from the according central pod annotation with the Downward API.
	ExpiredAt = RegisterSetting("ROX_EXPIRED_AT", AllowEmpty())
)
