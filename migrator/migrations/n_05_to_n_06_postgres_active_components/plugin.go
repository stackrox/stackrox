package n5ton6

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

// ComposeID creates an active component id from a deployment id and a component id
func ComposeID(deploymentID, componentID string) string {
	return fmt.Sprintf("%s:%s", deploymentID, componentID)
}

func convertActiveVuln(imageOsMap map[string]string, ac *storage.ActiveComponent) []*storage.ActiveComponent {
	edge, err := edges.FromString(ac.GetComponentId())
	if err != nil {
		log.WriteToStderrf("unexpected component id %q", ac.GetDeploymentId())
	}
	componentName := edge.ParentID
	componentVersion := edge.ChildID
	osToContext := make(map[string][]*storage.ActiveComponent_ActiveContext)
	for _, context := range ac.GetActiveContextsSlice() {
		os := imageOsMap[context.GetImageId()]
		osToContext[os] = append(osToContext[os], context)
	}
	ret := make([]*storage.ActiveComponent, 0, len(osToContext))
	for os, contexts := range osToContext {
		cloned := ac.Clone()
		cloned.ComponentId = pgSearch.IDFromPks([]string{componentName, componentVersion, os})
		cloned.Id = ComposeID(cloned.GetDeploymentId(), cloned.GetComponentId())
		cloned.ActiveContextsSlice = contexts
		ret = append(ret, cloned)
	}
	return ret
}
