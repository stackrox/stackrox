package clusters

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
)

func init() {
	deployers[v1.ClusterType_OPENSHIFT_CLUSTER] = newOpenshift()
}

type openshift struct{}

func newOpenshift() Deployer {
	return &openshift{}
}

func (o *openshift) Render(c Wrap) ([]*v1.File, error) {
	var openshiftParams *v1.OpenshiftParams
	clusterOpenshift, ok := c.OrchestratorParams.(*v1.Cluster_Openshift)
	if ok {
		openshiftParams = clusterOpenshift.Openshift
	}

	fields := fieldsFromWrap(c)
	addCommonKubernetesParams(openshiftParams.GetParams(), fields)
	fields["OpenshiftAPIEnv"] = env.OpenshiftAPI.EnvVar()
	fields["OpenshiftAPI"] = `"true"`

	filenames := []string{
		"kubernetes/sensor.yaml",
		"openshift/delete-sensor.sh",
		"openshift/sensor.sh",
		"openshift/sensor-image-setup.sh",
		"openshift/sensor-rbac.yaml",
	}

	return renderFilenames(filenames, fields)
}
