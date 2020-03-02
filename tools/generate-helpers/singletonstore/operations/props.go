package operations

// GeneratorProperties contains the values used by the generator to generate singleton-store classes.
type GeneratorProperties struct {
	Pkg                string
	Object             string
	HumanName          string
	BucketName         string
	AddInsteadOfUpsert bool
}
