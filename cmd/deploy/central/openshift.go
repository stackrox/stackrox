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
		"kubernetes/np.yaml",

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

func (o *openshift) Instructions() string {
	return `To deploy:
  1. Unzip the deployment bundle.
  2. Run image-setup.sh.
  3. Run central.sh.
  4. If you want to run the StackRox Clairify scanner, run clairify.sh.
  5. Expose Central:
       a. Using a Route:        ./route-setup.sh
       b. Using a NodePort:     oc create -f np.yaml
       c. Using a port forward: ./port-forward.sh 8443`
}
