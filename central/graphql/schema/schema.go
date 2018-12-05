package schema

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/graphql/generator"
	"github.com/stackrox/rox/generated/api/v1"
)

var (
	extraResolvers = make(map[string][]string)

	// WalkParameters is a set of shared parameters used for both
	// schema generation and resolver code generation
	WalkParameters = generator.TypeWalkParameters{
		IncludedTypes: []reflect.Type{
			reflect.TypeOf((*v1.Alert)(nil)),
			reflect.TypeOf((*v1.ListAlert)(nil)),
			reflect.TypeOf((*v1.Cluster)(nil)),
			reflect.TypeOf((*v1.Deployment)(nil)),
			reflect.TypeOf((*v1.ListDeployment)(nil)),
			reflect.TypeOf((*v1.Group)(nil)),
			reflect.TypeOf((*v1.Image)(nil)),
			reflect.TypeOf((*v1.ListImage)(nil)),
			reflect.TypeOf((*v1.Metadata)(nil)),
			reflect.TypeOf((*v1.NetworkFlow)(nil)),
			reflect.TypeOf((*v1.Node)(nil)),
			reflect.TypeOf((*v1.Notifier)(nil)),
			reflect.TypeOf((*v1.ProcessNameGroup)(nil)),
			reflect.TypeOf((*v1.Role)(nil)),
			reflect.TypeOf((*v1.SearchResult)(nil)),
			reflect.TypeOf((*v1.Secret)(nil)),
			reflect.TypeOf((*v1.ListSecret)(nil)),
			reflect.TypeOf((*v1.TokenMetadata)(nil)),
			reflect.TypeOf((*v1.GenerateTokenResponse)(nil)),
		},
	}
)

// AddResolver registers a GraphQL resolver on the specified message type.
// The resolver needs to be implemented as a method on the matching resolver struct.
func AddResolver(message proto.Message, resolver string) {
	n := reflect.TypeOf(message).Elem().Name()
	extraResolvers[n] = append(extraResolvers[n], resolver)
}

// AddQuery registers a GraphQL resolver on the Query object. The resolver needs
// to be implemented as a method on the root Resolver struct.
func AddQuery(resolver string) {
	extraResolvers["Query"] = append(extraResolvers["Query"], resolver)
}

// Schema returns the generated GraphQL schema
func Schema() string {
	return generator.GenerateSchema(WalkParameters, extraResolvers)
}
