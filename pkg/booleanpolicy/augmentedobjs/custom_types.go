package augmentedobjs

// This block enumerates custom tags.
const (
	DockerfileLineCustomTag      = "Dockerfile Line"
	ComponentAndVersionCustomTag = "Component And Version"
	NotInBaselineCustomTag       = "Not In Baseline"
	ContainerNameCustomTag       = "Container Name"
	ImageScanCustomTag           = "Image Scan"
	EnvironmentVarCustomTag      = "Environment Variable"
)

type dockerfileLine struct {
	Line string `search:"Dockerfile Line"`
}

type componentAndVersion struct {
	ComponentAndVersion string `search:"Component And Version"`
}

type baselineResult struct {
	NotInBaseline bool `search:"Not In Baseline"`
}

type envVar struct {
	EnvVar string `search:"Environment Variable"`
}
