package central

import (
	"github.com/stackrox/rox/generated/api/v1"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
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

	var err error
	c.K8sConfig.Registry, err = kubernetesPkg.GetResolvedRegistry(c.K8sConfig.PreventImage)
	if err != nil {
		return nil, err
	}

	filenames := []string{
		"kubernetes/central.yaml",
		"kubernetes/ca-setup.sh",
		"kubernetes/delete-ca.sh",
		"kubernetes/np.yaml",

		"openshift/central.sh",
		"openshift/central-rbac.yaml",
		"openshift/clairify.sh",
		"openshift/clairify.yaml",
		"openshift/image-setup.sh",
		"openshift/port-forward.sh",
		"openshift/route-setup.sh",
	}

	return renderFilenames(filenames, &c, "/data/assets/docker-auth.sh")
}

func (o *openshift) Instructions() string {
	return `To deploy:
  1. Unzip the deployment bundle.
  2. If you need to add additional trusted CAs, run ca-setup.sh.
  3. Run image-setup.sh.
  4. Run central.sh.
  5. If you want to run the StackRox Clairify scanner, run clairify.sh.
  6. Expose Central:
       a. Using a Route:        ./route-setup.sh
       b. Using a NodePort:     oc create -f np.yaml
       c. Using a port forward: ./port-forward.sh 8443`
}
