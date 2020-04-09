package features

var (
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

	// RefreshTokens enables supports for refresh tokens & OIDC code flow.
	RefreshTokens = registerFeature("Refresh tokens", "ROX_REFRESH_TOKENS", true)

	// ImageLabelPolicy enables the Required Image Label policy type
	ImageLabelPolicy = registerFeature("Enable the Required Image Label Policy", "ROX_REQUIRED_IMAGE_LABEL_POLICY", true)

	// AdmissionControlService enables running admission control as a separate microservice.
	AdmissionControlService = registerFeature("Separate admission control microservice", "ROX_ADMISSION_CONTROL_SERVICE", true)

	// PodDeploymentSeparation enables support for tracking pods and deployments separately
	PodDeploymentSeparation = registerFeature("Separate Pods and Deployments", "ROX_POD_DEPLOY_SEPARATE", true)

	// AdmissionControlEnforceOnUpdate enables support for having the admission controller enforce on updates.
	AdmissionControlEnforceOnUpdate = registerFeature("Allow admission controller to enforce on update", "ROX_ADMISSION_CONTROL_ENFORCE_ON_UPDATE", true)

	// DryRunPolicyJobMechanism enables submitting dry run of a policy as a job, and querying the status using job id.
	DryRunPolicyJobMechanism = registerFeature("Dry run policy job mechanism", "ROX_DRY_RUN_JOB", false)

	// RocksDB enables using RocksDB as a databases instead of BadgerDB
	RocksDB = registerFeature("Use RocksDB instead of BadgerDB", "ROX_ROCKSDB", false)

	// BooleanPolicyLogic enables support for an extended policy logic
	BooleanPolicyLogic = registerFeature("Enable Boolean Policy Logic", "ROX_BOOLEAN_POLICY_LOGIC", false)

	// PolicyImportExport feature flag enables policy import and export
	PolicyImportExport = registerFeature("Enable Import/Export for Analyst Workflow", "ROX_POLICY_IMPORT_EXPORT", false)
)
