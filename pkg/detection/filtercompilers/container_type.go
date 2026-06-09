package filtercompilers

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
)

func init() {
	booleanpolicy.RegisterFilterPlugin[[]storage.SkipContainerType](
		containerTypePlugin,
		map[booleanpolicy.MatcherKind]booleanpolicy.ContextExtractor{
			// Only deployment-context matchers have containers to filter.
			booleanpolicy.DeploymentKind:  extractContainersFromDeployment,
			booleanpolicy.ProcessKind:     extractContainersFromDeployment,
			booleanpolicy.NetworkFlowKind: extractContainersFromDeployment,
			booleanpolicy.FileAccessKind:  extractContainersFromDeployment,
		},
	)
}

// extractContainersFromDeployment extracts the containers slice from an EnhancedDeployment context.
func extractContainersFromDeployment(matchData interface{}) interface{} {
	return matchData.(booleanpolicy.EnhancedDeployment).Deployment.GetContainers()
}

// containerTypePlugin owns the EvaluationFilter.skip_container_types field.
// Returns nil when skip_container_types is empty — no filtering needed.
// The returned factory receives []*storage.Container already extracted by the ContextExtractor.
func containerTypePlugin(f *storage.EvaluationFilter) booleanpolicy.ValueFilterFactory {
	skipTypes := buildContainerSkipSet(f.GetSkipContainerTypes())
	if len(skipTypes) == 0 {
		return nil
	}
	return func(data interface{}) pathutil.ValueFilter {
		containers, ok := data.([]*storage.Container)
		if !ok || len(containers) == 0 {
			return nil
		}
		return containerTypeValueFilter(skipTypes)
	}
}

// containerTypeValueFilter returns a ValueFilter that pre-terminates Containers[i]
// when the container type is in the skip set. Filtering at the container level means
// the container's entire subtree — images, components, layers — is not evaluated.
// All non-container element types pass through unchanged.
func containerTypeValueFilter(skipTypes map[storage.ContainerType]struct{}) pathutil.ValueFilter {
	return func(v pathutil.AugmentedValue, i int) bool {
		elem := v.Underlying().Index(i).Interface()
		container, ok := elem.(*storage.Container)
		if !ok {
			// Not at the Containers slice — pass through.
			return true
		}
		_, shouldSkip := skipTypes[container.GetType()]
		return !shouldSkip
	}
}

// buildContainerSkipSet converts SkipContainerType values into a set of ContainerType
// values to exclude during evaluation.
func buildContainerSkipSet(skipTypes []storage.SkipContainerType) map[storage.ContainerType]struct{} {
	if len(skipTypes) == 0 {
		return nil
	}
	set := make(map[storage.ContainerType]struct{}, len(skipTypes))
	for _, s := range skipTypes {
		set[skipContainerTypeToContainerType(s)] = struct{}{}
	}
	return set
}

// skipContainerTypeToContainerType maps a SkipContainerType filter value to the
// ContainerType it represents in storage.Container.
func skipContainerTypeToContainerType(s storage.SkipContainerType) storage.ContainerType {
	switch s {
	case storage.SkipContainerType_SKIP_INIT:
		return storage.ContainerType_INIT
	}
	return storage.ContainerType_REGULAR
}
