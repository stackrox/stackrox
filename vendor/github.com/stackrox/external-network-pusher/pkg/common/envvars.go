package common

var (
	verbose = false
)

// Verbose returns if verbose options is set
func Verbose() bool {
	return verbose
}

// SetVerbose enables verbose mode
func SetVerbose() {
	verbose = true
}
