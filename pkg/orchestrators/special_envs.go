package orchestrators

// SpecialEnvVar refers to special environment variables with auto-populated values.
type SpecialEnvVar string

const (
	// NodeName is the special environment variable that will expand to the (orchestrator) node name.
	NodeName SpecialEnvVar = "ROX_NODE_NAME"
)
