package central

import (
	"github.com/stackrox/rox/generated/api/v1"
)

func init() {
	Deployers[v1.ClusterType_OPENSHIFT_CLUSTER] = newOpenshift()
}

type openshift struct{}

func newOpenshift() deployer {
	return &openshift{}
}

func (o *openshift) Render(c Config) ([]*v1.File, error) {
	injectImageTags(&c)

	filenames := []string{
		"kubernetes/central.yaml",

		"openshift/central.sh",
		"openshift/central-rbac.yaml",
		"openshift/clairify.sh",
		"openshift/clairify.yaml",
		"openshift/image-setup.sh",
		"openshift/port-forward.sh",
		"openshift/route-setup.sh",
	}

	return renderFilenames(filenames, c)
}
