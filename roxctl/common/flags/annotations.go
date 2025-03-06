package flags

const (
	// NoInteractiveKey can be used to mark a flag as to not be prompted for in the interactive installer.
	NoInteractiveKey = "no-interactive"
	// OptionalKey can be used to mark a flag as optional.
	OptionalKey = "optional"
	// MandatoryKey can be used to mark a flag as mandatory in the interactive installer, meaning that
	// when prompted for a value, the user must enter a non-empty string.
	MandatoryKey = "mandatory"
	// DependenciesKey can be used to mark that a flag depends on other flags, with the
	// effect that if any of the other flags is empty/unset, the prompt for this flag will be
	// skipped.
	DependenciesKey = "dependencies"
	// InteractiveUsageKey allows setting a different `usage` string for interactive prompts.
	InteractiveUsageKey = "interactive-usage"
	// PasswordKey allows an echoless prompt.
	PasswordKey = "password"
)

var (
	// OutputDirManuallySet returns true iff --output-dir was manually set in the roxctl command
	OutputDirManuallySet = false
)
