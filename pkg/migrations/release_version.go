package migrations

//go:generate go run ../../tools/generate-helpers/release-versions

// ReleaseVersion records a stream's initial GA database sequence.
type ReleaseVersion struct {
	Version  string
	Sequence int
}
