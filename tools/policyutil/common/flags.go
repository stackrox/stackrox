package common

// Flag names used by various commands.
const (
	VerboseFlagName      = "verbose"
	VerboseFlagShorthand = "v"

	InteractiveFlagName      = "interactive"
	InteractiveFlagShorthand = "i"

	FileFlagName      = "file"
	FileFlagShorthand = "f"

	DirectoryFlagName      = "dir"
	DirectoryFlagShorthand = "d"

	OutputFlagName      = "out"
	OutputFlagShorthand = "o"

	DryRunFlagName      = "dry-run"
	DryRunFlagShorthand = "n"

	ReadOnlyFlagName = "ensure-read-only"
)

// Allowed values for read-only flag.
const (
	None     readOnlyPolicySettings = "none"
	Mitre    readOnlyPolicySettings = "mitre"
	Criteria readOnlyPolicySettings = "criteria"
)

var (
	// Verbose indicates whether we run in the verbose mode.
	Verbose bool
	// Interactive indicates whether we run in the interactive mode.
	Interactive bool

	// ReadOnlySettingStrToType maps strings to readOnlyPolicySettings type.
	ReadOnlySettingStrToType = map[string]readOnlyPolicySettings{
		"none":     None,
		"mitre":    Mitre,
		"criteria": Criteria,
	}
)

type readOnlyPolicySettings string

func (s readOnlyPolicySettings) String() string {
	return string(s)
}
