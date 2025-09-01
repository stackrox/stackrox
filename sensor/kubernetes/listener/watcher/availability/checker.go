package availability

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sapi"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/listener/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type crdWatcher interface {
	AddResourceToWatch(string) error
}

type Checker interface {
	Available(client.Interface) (bool, error)
	AppendToCRDWatcher(crdWatcher) error
	GetResources() []k8sapi.APIResource
}

type checker struct {
	gv        schema.GroupVersion
	resources []k8sapi.APIResource
}

// NewChecker creates a new availability checker
func NewChecker(gv schema.GroupVersion, resources []k8sapi.APIResource) *checker {
	return &checker{
		gv:        gv,
		resources: resources,
	}
}

// GetResources returns the resources for which the availability needs to be checked.
func (c *checker) GetResources() []k8sapi.APIResource {
	return c.resources
}

// Available returns 'true' if all configured resources are available in the cluster
func (c *checker) Available(client client.Interface) (bool, error) {
	var resourceList *v1.APIResourceList
	var err error
	if resourceList, err = utils.ServerResourcesForGroup(client, c.gv.String()); err != nil {
		return false, errors.Wrapf(err, "Checking API resources for group %q", c.gv.String())
	}
	for _, r := range c.resources {
		if !utils.ResourceExists(resourceList, r.Name, c.gv.String()) {
			return false, nil
		}
	}
	return true, nil
}

// AppendToCRDWatcher adds the selected resources to the CRD watcher
func (c *checker) AppendToCRDWatcher(watcher crdWatcher) error {
	for _, r := range c.resources {
		nameGroupString := apiResourceToNameGroupString(r)
		if err := watcher.AddResourceToWatch(nameGroupString); err != nil {
			return errors.Wrapf(err, "watching resource name=%q group=%q version=%q", r.Name, r.Group, r.Version)
		}
	}
	return nil
}

func apiResourceToNameGroupString(resource k8sapi.APIResource) string {
	return fmt.Sprintf("%s.%s", resource.Name, resource.Group)
}
