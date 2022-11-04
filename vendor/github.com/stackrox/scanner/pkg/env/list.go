package env

var (
	// LanguageVulns enables language vulnerabilities.
	LanguageVulns = RegisterBooleanSetting("ROX_LANGUAGE_VULNS", true, AllowWithoutRox())

	// SkipPeerValidation skips peer certificate validation (typically used for testing).
	// When disabled, only Central ingress is allowed, by default. See SlimMode and
	// OpenshiftAPI for other ingress controls.
	SkipPeerValidation = RegisterBooleanSetting("ROX_SKIP_PEER_VALIDATION", false)

	// SlimMode enables slim-mode. When enabled, Scanner only supports a subset of APIs,
	// and only Sensor ingress is allowed.
	// If SkipPeerValidation or OpenshiftAPI is enabled, the ingress implications are ignored.
	SlimMode = RegisterBooleanSetting("ROX_SLIM_MODE", false)

	// OpenshiftAPI indicates Scanner is running in an OpenShift environment.
	// When set to "true", ingress is allowed from both Central and Sensor.
	// This is ignored if SkipPeerValidation is enabled.
	// This variable was copied over from the stackrox repo.
	OpenshiftAPI = RegisterBooleanSetting("ROX_OPENSHIFT_API", false)
)
