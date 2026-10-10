package migrations

//go:generate go run ../../tools/generate-helpers/release-versions

// ReleaseVersion records a stream's initial-release database sequence (GA, or latest RC until GA).
type ReleaseVersion struct {
	Version  string
	Sequence int
}
