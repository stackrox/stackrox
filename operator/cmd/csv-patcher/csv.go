package main

// csvDocument represents the ClusterServiceVersion YAML structure
// We only define fields we need to modify, using map[string]interface{} for the rest
type csvDocument struct {
	Metadata struct {
		Name        string                 `yaml:"name"`
		Annotations map[string]interface{} `yaml:"annotations"`
		Labels      map[string]interface{} `yaml:"labels,omitempty"`
	} `yaml:"metadata"`
	Spec struct {
		Version       string                   `yaml:"version"`
		Replaces      string                   `yaml:"replaces,omitempty"`
		Skips         []string                 `yaml:"skips,omitempty"`
		RelatedImages []map[string]interface{} `yaml:"relatedImages,omitempty"`
		CustomResourceDefinitions struct {
			Owned []map[string]interface{} `yaml:"owned"`
		} `yaml:"customresourcedefinitions"`
		// Keep other fields as-is
		Rest map[string]interface{} `yaml:",inline"`
	} `yaml:"spec"`
}

// relatedImage represents an entry in spec.relatedImages
type relatedImage struct {
	Name  string `yaml:"name"`
	Image string `yaml:"image"`
}
