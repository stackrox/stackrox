package resources

import (
	"strings"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

var (
	log = logging.LoggerForModule()
)

// Metadata represents Kubernetes API resource metadata.
type Metadata struct {
	v1.APIResource
	Purpose Purpose
}

// GroupVersionKind returns the `schema.GroupVersionKind` of an API resource. The returned value is safe to be used
// in map keys etc.
func (m *Metadata) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   m.Group,
		Version: m.Version,
		Kind:    m.Kind,
	}
}

// GroupVersionResource returns the `schema.GroupVersionResource` of an API resource.
func (m *Metadata) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    m.Group,
		Version:  m.Version,
		Resource: m.Name,
	}
}

// String returns a string representation for this resource.
func (m *Metadata) String() string {
	gvr := m.GroupVersionResource()
	return gvr.String()
}

func populateFromResourceList(resourceList *v1.APIResourceList, expectedGVKs map[schema.GroupVersionKind]struct{}, outputMap *map[schema.GroupVersionKind]*Metadata) *schema.GroupVersion {
	if resourceList == nil || len(resourceList.APIResources) == 0 {
		return nil
	}
	gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
	if err != nil {
		// Should never happen, but let's be forgiving if it does.
		log.Warnf("Failed to parse group version for resource list %v: %v", resourceList, utils.ShouldErr(err))
		return nil
	}

	for _, apiResource := range resourceList.APIResources {
		if strings.ContainsRune(apiResource.Name, '/') {
			continue // ignore sub-resources like `deployments/scale`
		}

		if apiResource.Group == "" {
			apiResource.Group = gv.Group
		}
		if apiResource.Version == "" {
			apiResource.Version = gv.Version
		}

		gvk := schema.GroupVersionKind{
			Group:   apiResource.Group,
			Version: apiResource.Version,
			Kind:    apiResource.Kind,
		}

		if _, alreadyExists := (*outputMap)[gvk]; alreadyExists {
			continue
		}

		if _, isExpectedGVK := expectedGVKs[gvk]; !isExpectedGVK {
			continue
		}

		md := &Metadata{
			APIResource: apiResource,
		}
		(*outputMap)[gvk] = md
	}

	return &gv
}

// GetAvailableResources uses the Kubernetes Discovery API to list all relevant resources on the server.
// It returns metadata for all the GVKs passed in expectedGVKs.
// It returns an error if it wasn't able to populate some of the expected GVKs due to an unexpected error.
// See: https://stack-rox.atlassian.net/browse/ROX-4429 as an example of why we are making this function
// so resilient, despite the added complexity introduced.
func GetAvailableResources(client discovery.ServerResourcesInterface, expectedGVKs map[schema.GroupVersionKind]struct{}) (map[schema.GroupVersionKind]*Metadata, error) {
	result := make(map[schema.GroupVersionKind]*Metadata)
	_, resourceLists, err := client.ServerGroupsAndResources()
	if err != nil {
		log.Warnf("Error retrieving list of server resources: %v. Continuing with partial results...", err)
	}

	seenGVs := make(map[schema.GroupVersion]struct{})
	for _, resourceList := range resourceLists {
		gv := populateFromResourceList(resourceList, expectedGVKs, &result)
		if gv != nil {
			seenGVs[*gv] = struct{}{}
		}
	}
	// If err is nil, then we've got everything that's available, just return.
	if err == nil {
		return result, nil
	}

	// If we got only partial results, let's make targeted API calls to fetch the missing ones,
	// from the GVKs we care about.
	missingGVs := make(map[schema.GroupVersion]struct{})
	for expectedGVK := range expectedGVKs {
		if _, found := result[expectedGVK]; !found {
			gv := expectedGVK.GroupVersion()
			// If we've seen the GV from the call to ClientGroupsAndResources, then no point
			// making a calling with it.
			if _, seen := seenGVs[gv]; !seen {
				missingGVs[gv] = struct{}{}
			}
		}
	}
	errorList := errorhelpers.NewErrorList("finding metadata for missing group versions")

	for gv := range missingGVs {
		resourceListForGroupVersion, err := client.ServerResourcesForGroupVersion(gv.String())
		if err != nil {
			// If the resource doesn't exist on the server, that's okay. This is a definitive response.
			// If we need to create the resource downstream, we will fatal out then, which is fine.
			if !k8sErrors.IsNotFound(err) {
				errorList.AddWrapf(err, "fetching resourceList for group version %s: %v", gv, err)
			}
			continue
		}
		populateFromResourceList(resourceListForGroupVersion, expectedGVKs, &result)
	}

	return result, errorList.ToError()
}
