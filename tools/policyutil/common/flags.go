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
)

var (
	// Verbose indicates whether we run in the verbose mode.
	Verbose bool
	// Interactive indicates whether we run in the interactive mode.
	Interactive bool
)
