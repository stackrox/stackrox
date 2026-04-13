# GraphQL Loader Init Migration

## Current State
- ~15 GraphQL loader files in central/graphql/resolvers/loaders/
- Each has init() calling RegisterTypeFactory()
- Loaders: cluster_cves, componentsV2, deployments, images, namespaces, etc.

## Migration Strategy
Phase 3.3 establishes initGraphQL() stub in central/app/init.go

Future work (separate PR):
1. Refactor loaders package to export registration function
2. Call registration from central/app/init.go initGraphQL()
3. Remove init() from all loader files

## Files affected:
- central/graphql/resolvers/loaders/cluster_cves.go
- central/graphql/resolvers/loaders/componentsV2.go
- central/graphql/resolvers/loaders/context.go
- central/graphql/resolvers/loaders/deployments.go
- central/graphql/resolvers/loaders/factory.go
- central/graphql/resolvers/loaders/image_cves_v2.go
- central/graphql/resolvers/loaders/images.go
- central/graphql/resolvers/loaders/images_v2.go
- central/graphql/resolvers/loaders/list_deployments.go
- central/graphql/resolvers/loaders/namespaces.go
- central/graphql/resolvers/loaders/node_components.go
- central/graphql/resolvers/loaders/node_cves.go
- central/graphql/resolvers/loaders/nodes.go
- central/graphql/resolvers/loaders/policies.go
- central/graphql/resolvers/loaders/utils.go

## RegisterTypeFactory Pattern
Each loader file follows this pattern:

```go
var deploymentLoaderType = reflect.TypeOf(storage.Deployment{})

func init() {
    RegisterTypeFactory(reflect.TypeOf(storage.Deployment{}), func() interface{} {
        return NewDeploymentLoader(datastore.Singleton(), deploymentsView.Singleton())
    })
}
```

## Future Migration Approach
1. Create central/graphql/resolvers/loaders/register.go with exported RegisterAllLoaders() function
2. Move all RegisterTypeFactory calls into RegisterAllLoaders()
3. Call RegisterAllLoaders() from central/app/init.go initGraphQL()
4. Remove init() functions from individual loader files
5. Verify GraphQL queries still work in integration tests
