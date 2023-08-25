package centralservices

import (
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
)

const (
	chartValues = `
allowNonstandardNamespace: true
licenseKey: "my license key"
env:
  platform: gke
  openshift: true
  istio: true
  proxyConfig: "proxy config"
imagePullSecrets:
  username: myuser
  password: mypass
central:
  persistence:
    none: true
  exposure:
    loadBalancer:
      enabled: true
`
)

var (
	allowedGlobalResources = []string{
		"/StorageClass/stackrox-gke-ssd",
	}
)

type multiNamespaceSuite struct {
	baseSuite
}

func TestMultiNamespace(t *testing.T) {
	suite.Run(t, new(multiNamespaceSuite))
}

// Extracts string-identifiers for all the Kubernetes resources in the provided map:
// For example, a deployment of name "central" in namespace "foo" would have the identifier "foo/Deployment/central".
// This makes the diffing easy.
func (s *multiNamespaceSuite) extractResourceIdentifiers(rendered map[string]string) set.StringSet {
	resourceIdentifiers := set.NewStringSet()
	for _, obj := range s.ParseObjects(rendered) {
		resourceIdentifiers.Add(fmt.Sprintf("%s/%s/%s", obj.GetNamespace(), obj.GetKind(), obj.GetName()))
	}
	for _, obj := range allowedGlobalResources {
		resourceIdentifiers.Remove(obj)
	}
	return resourceIdentifiers
}

func (s *multiNamespaceSuite) TestDisjointResourcesCustom() {
	_, renderedResourcesFoo := s.LoadAndRenderWithNamespace("foo", chartValues)
	resourceIdentifiersFoo := s.extractResourceIdentifiers(renderedResourcesFoo)

	_, renderedResourcesBar := s.LoadAndRenderWithNamespace("bar", chartValues)
	resourceIdentifiersBar := s.extractResourceIdentifiers(renderedResourcesBar)

	intersection := resourceIdentifiersFoo.Intersect(resourceIdentifiersBar)

	// Print differences between the two sets in case of test failure.
	for obj := range intersection {
		fmt.Fprintf(os.Stderr, "Resource name overlap: %s\n", obj)
	}

	// Check if the resource identifiers are disjoint.
	s.Require().True(intersection.IsEmpty())
}

func (s *multiNamespaceSuite) TestDisjointResourcesStandardAndCustom() {
	_, renderedResourcesStackrox := s.LoadAndRenderWithNamespace("stackrox", chartValues)
	resourceIdentifiersStackrox := s.extractResourceIdentifiers(renderedResourcesStackrox)

	_, renderedResourcesFoo := s.LoadAndRenderWithNamespace("foo", chartValues)
	resourceIdentifiersFoo := s.extractResourceIdentifiers(renderedResourcesFoo)

	intersection := resourceIdentifiersStackrox.Intersect(resourceIdentifiersFoo)

	// Print differences between the two sets in case of test failure.
	for obj := range intersection {
		fmt.Fprintf(os.Stderr, "Resource name overlap: %s\n", obj)
	}

	// Check if the resource identifiers are disjoint.
	s.Require().True(intersection.IsEmpty())
}
