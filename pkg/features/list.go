package features

var (
	// ConfigMgmtUI enables the config management UI.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	ConfigMgmtUI = registerFeature("Enable Config Mgmt UI", "ROX_CONFIG_MGMT_UI", true)

	// VulnMgmtUI enables the vulnerability management UI.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	VulnMgmtUI = registerFeature("Enable Vulnerability Management UI", "ROX_VULN_MGMT_UI", true)

	// Dackbox enables the id graph layer on top of badger.
	Dackbox = registerFeature("Use DackBox layer for the embedded Badger DB", "ROX_DACKBOX", true)

	// Telemetry enables the telemetry features
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	Telemetry = registerFeature("Enable support for telemetry", "ROX_TELEMETRY", true)

	// DiagnosticBundle enables support for obtaining extended diagnostic information.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	DiagnosticBundle = registerFeature("Enable support for diagnostic bundle download", "ROX_DIAGNOSTIC_BUNDLE", true)

	// AnalystNotesUI enables the Analyst Notes UI.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	AnalystNotesUI = registerFeature("Enable Analyst Notes UI", "ROX_ANALYST_NOTES_UI", false)

	// EventTimelineUI enables the Event Timeline UI.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	EventTimelineUI = registerFeature("Enable Event Timeline UI", "ROX_EVENT_TIMELINE_UI", false)

	// NistSP800_53 enables the NIST SP 800-53 compliance standard.
	NistSP800_53 = registerFeature("NIST SP 800-53", "ROX_NIST_800_53", true)

	// RefreshTokens enables supports for refresh tokens & OIDC code flow.
	RefreshTokens = registerFeature("Refresh tokens", "ROX_REFRESH_TOKENS", true)

	// ImageLabelPolicy enables the Required Image Label policy type
	ImageLabelPolicy = registerFeature("Enable the Required Image Label Policy", "ROX_REQUIRED_IMAGE_LABEL_POLICY", true)

	// AdmissionControlService enables running admission control as a separate microservice.
	AdmissionControlService = registerFeature("Separate admission control microservice", "ROX_ADMISSION_CONTROL_SERVICE", false)

	// PodDeploymentSeparation enables support for tracking pods and deployments separately
	PodDeploymentSeparation = registerFeature("Separate Pods and Deployments", "ROX_POD_DEPLOY_SEPARATE", false)

	// AdmissionControlEnforceOnUpdate enables support for having the admission controller enforce on updates.
	AdmissionControlEnforceOnUpdate = registerFeature("Allow admission controller to enforce on update", "ROX_ADMISSION_CONTROL_ENFORCE_ON_UPDATE", false)
)
