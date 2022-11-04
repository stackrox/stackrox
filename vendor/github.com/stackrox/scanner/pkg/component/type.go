package component

// SourceType represents the specific type of a language-level component.
//
//go:generate stringer -type=SourceType
type SourceType int

// This block enumerates valid types.
const (
	UnsetSourceType SourceType = iota
	GemSourceType
	JavaSourceType
	NPMSourceType
	PythonSourceType
	DotNetCoreRuntimeSourceType

	SentinelEndSourceType
)
