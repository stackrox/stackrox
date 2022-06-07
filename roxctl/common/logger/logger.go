package logger

// Logger is a struct responsible for printing messages. It should be preferred over fmt functions.
type Logger interface {
	// ErrfLn prints a formatted string with a newline, prefixed with ERROR and colorized
	ErrfLn(format string, a ...interface{})

	// WarnfLn prints a formatted string with a newline, prefixed with WARN and colorized
	WarnfLn(format string, a ...interface{})

	// InfofLn prints a formatted string with a newline, prefixed with INFO and colorized
	InfofLn(format string, a ...interface{})

	// PrintfLn prints a formatted string with newline at the end
	PrintfLn(format string, a ...interface{})
}
