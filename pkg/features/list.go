package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

var (
	// AnalystNotesUI enables the Analyst Notes UI.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	AnalystNotesUI = registerFeature("Enable Analyst Notes UI", "ROX_ANALYST_NOTES_UI", true)

	// EventTimelineClusteredEventsUI enables the Event Timeline UI for Clustered Events.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	EventTimelineClusteredEventsUI = registerFeature("Enable Event Timeline Clustered Events UI", "ROX_EVENT_TIMELINE_CLUSTERED_EVENTS_UI", true)

	// ImageLabelPolicy enables the Required Image Label policy type
	ImageLabelPolicy = registerFeature("Enable the Required Image Label Policy", "ROX_REQUIRED_IMAGE_LABEL_POLICY", true)

	// AdmissionControlService enables running admission control as a separate microservice.
	AdmissionControlService = registerFeature("Separate admission control microservice", "ROX_ADMISSION_CONTROL_SERVICE", true)

	// AdmissionControlEnforceOnUpdate enables support for having the admission controller enforce on updates.
	AdmissionControlEnforceOnUpdate = registerFeature("Allow admission controller to enforce on update", "ROX_ADMISSION_CONTROL_ENFORCE_ON_UPDATE", true)

	// PolicyImportExport feature flag enables policy import and export
	PolicyImportExport = registerFeature("Enable Import/Export for Analyst Workflow", "ROX_POLICY_IMPORT_EXPORT", true)

	// AuthTestMode feature flag allows test mode flow for new auth provider in UI
	AuthTestMode = registerFeature("Enable Auth Test Mode UI", "ROX_AUTH_TEST_MODE_UI", true)

	// CurrentUserInfo enables showing information about the current user in UI
	CurrentUserInfo = registerFeature("Enable Current User Info UI", "ROX_CURRENT_USER_INFO", true)

	// ComplianceInNodes enables running of node-related Compliance checks in the compliance pods
	ComplianceInNodes = registerFeature("Enable compliance checks in nodes", "ROX_COMPLIANCE_IN_NODES", true)

	// RocksDB enables running of RocksDB
	RocksDB = registerFeature("Runs RocksDB instead of BadgerDB", "ROX_ROCKSDB", true)

	// csvExport enables CSV export of search results.
	csvExport = registerFeature("Enable CSV export of search results", "ROX_CSV_EXPORT", false)

	// ClusterHealthMonitoring enables monitoring of sensor and collector health
	ClusterHealthMonitoring = registerFeature("Enable cluster health monitoring", "ROX_CLUSTER_HEALTH_MONITORING", false)
)
