package clusters

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/zip"
)

func init() {
	deployers[storage.ClusterType_OPENSHIFT_CLUSTER] = newOpenshift()
}

type openshift struct{}

func newOpenshift() Deployer {
	return &openshift{}
}

func (o *openshift) Render(c Wrap) ([]*zip.File, error) {
	var openshiftParams *storage.OpenshiftParams
	clusterOpenshift, ok := c.OrchestratorParams.(*storage.Cluster_Openshift)
	if ok {
		openshiftParams = clusterOpenshift.Openshift
	}

	fields, err := fieldsFromWrap(c)
	if err != nil {
		return nil, err
	}
	addCommonKubernetesParams(openshiftParams.GetParams(), fields)
	fields["OpenshiftAPIEnv"] = env.OpenshiftAPI.EnvVar()

	filenames := []string{
		"kubernetes/kubectl/sensor.yaml",
		"openshift/kubectl/sensor.sh",
		"openshift/kubectl/sensor-image-setup.sh",
		"openshift/kubectl/sensor-rbac.yaml",
	}

	if c.MonitoringEndpoint != "" {
		filenames = append(filenames, monitoringFilenames...)
	}

	return renderFilenames(filenames, fields, "/data/assets/docker-auth.sh")
}
