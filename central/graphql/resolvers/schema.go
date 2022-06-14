package resolvers

import (
	"github.com/stackrox/rox/central/graphql/generator"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	builderOnce     sync.Once
	builderInstance generator.SchemaBuilder
)

func getBuilder() generator.SchemaBuilder {
	builderOnce.Do(func() {
		builderInstance = generator.NewSchemaBuilder()
		registerGeneratedTypes(builderInstance)
	})
	return builderInstance
}

// Schema outputs the generated schema from the package level state
func Schema() string {
	s, err := builderInstance.Render()
	if err == nil {
		return s
	}
	panic(err)
}
