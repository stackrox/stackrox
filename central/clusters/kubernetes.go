package clusters

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
)

func init() {
	deployers[v1.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
}

type kubernetes struct{}

func newKubernetes() Deployer {
	return &kubernetes{}
}

func addCommonKubernetesParams(params *v1.CommonKubernetesParams, fields map[string]interface{}) {
	fields["Namespace"] = params.GetNamespace()
	fields["NamespaceEnv"] = env.Namespace.EnvVar()
}

var monitoringFilenames = []string{
	"kubernetes/kubectl/telegraf.conf",
}

func (k *kubernetes) Render(c Wrap) ([]*v1.File, error) {
	var kubernetesParams *v1.KubernetesParams
	clusterKube, ok := c.OrchestratorParams.(*v1.Cluster_Kubernetes)
	if ok {
		kubernetesParams = clusterKube.Kubernetes
	}

	fields, err := fieldsFromWrap(c)
	if err != nil {
		return nil, err
	}
	addCommonKubernetesParams(kubernetesParams.GetParams(), fields)

	filenames := []string{
		"kubernetes/kubectl/sensor.sh",
		"kubernetes/kubectl/sensor.yaml",
		"kubernetes/kubectl/sensor-rbac.yaml",
		"kubernetes/kubectl/delete-sensor.sh",
	}

	if c.MonitoringEndpoint != "" {
		filenames = append(filenames, monitoringFilenames...)
	}

	return renderFilenames(filenames, fields, "/data/assets/docker-auth.sh")
}
