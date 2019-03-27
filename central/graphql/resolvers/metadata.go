package resolvers

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("metadata: Metadata"),
	)
}

// Metadata returns a metadata object containing the stackrox version.
func (resolver *Resolver) Metadata() (*metadataResolver, error) {
	ver := version.GetMainVersion()
	return resolver.wrapMetadata(&v1.Metadata{Version: ver}, ver != "", nil)
}
