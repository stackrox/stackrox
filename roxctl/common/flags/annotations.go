package flags

const (
	// OptionalKey can be used to mark a flag as optional.
	OptionalKey = "optional"
	// DependenciesKey can be used to mark that a flag depends on other flags, with the
	// effect that if any of the other flags is empty/unset, the prompt for this flag will be
	// skipped.
	DependenciesKey = "dependencies"
	// InteractiveUsageKey allows setting a different `usage` string for interactive prompts.
	InteractiveUsageKey = "mode-usage"
)
