package resolvers

import (
	"github.com/stackrox/rox/central/graphql/schema"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/version"
)

func init() {
	schema.AddQuery("metadata: Metadata")
}

// Metadata returns a metadata object containing the stackrox version.
func (resolver *Resolver) Metadata() (*metadataResolver, error) {
	ver, err := version.GetVersion()
	return resolver.wrapMetadata(&v1.Metadata{Version: ver}, ver != "", err)
}
