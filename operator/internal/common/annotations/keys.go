package annotations

const (
	// ConfigHashAnnotation is a level-based annotation that, when changed on the CR,
	// is propagated by a PostRenderer onto pod templates to drive restarts.
	ConfigHashAnnotation = "app.stackrox.io/config-hash"
)
