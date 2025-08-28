package availability

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sapi"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/listener/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	log = logging.LoggerForModule()
)

type checker struct {
	gv        schema.GroupVersion
	resources []k8sapi.APIResource
}

// New Creates a new availability checker
func New(gv schema.GroupVersion, resources []k8sapi.APIResource) *checker {
	return &checker{
		gv:        gv,
		resources: resources,
	}
}

// GetResources returns the resources, those which the availability needs to be checked
func (c *checker) GetResources() []k8sapi.APIResource {
	return c.resources
}

// Available returns 'true' if the configured resources are available in the cluster
func (c *checker) Available(client client.Interface) bool {
	var resourceList *v1.APIResourceList
	var err error
	if resourceList, err = utils.ServerResourcesForGroup(client, c.gv.String()); err != nil {
		log.Errorf("Checking API resources for group %q: %v", c.gv.String(), err)
		return false
	}
	for _, r := range c.resources {
		if !utils.ResourceExists(resourceList, r.Name, c.gv.String()) {
			return false
		}
	}
	return true
}

type CrdWatcher interface {
	AddResourceToWatch(string) error
}

// AppendToCRDWatcher adds the Compliance Operator resources to the CRD watcher
func (c *checker) AppendToCRDWatcher(watcher CrdWatcher) error {
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
