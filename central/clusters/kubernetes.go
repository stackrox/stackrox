package clusters

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
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
}

func (k *kubernetes) Render(c Wrap) ([]*v1.File, error) {
	var kubernetesParams *v1.KubernetesParams
	clusterKube, ok := c.OrchestratorParams.(*v1.Cluster_Kubernetes)
	if ok {
		kubernetesParams = clusterKube.Kubernetes
	}

	fields := fieldsFromWrap(c)
	addCommonKubernetesParams(kubernetesParams.GetParams(), fields)

	fields["OpenshiftAPIEnv"] = env.OpenshiftAPI.EnvVar()
	fields["OpenshiftAPI"] = `"false"`

	fields["ImagePullSecretEnv"] = env.ImagePullSecrets.EnvVar()
	fields["ImagePullSecret"] = kubernetesParams.GetImagePullSecret()

	var err error
	fields["Registry"], err = kubernetesPkg.GetResolvedRegistry(c.PreventImage)
	if err != nil {
		return nil, err
	}

	filenames := []string{
		"kubernetes/sensor.sh",
		"kubernetes/sensor.yaml",
		"kubernetes/sensor-rbac.yaml",
		"kubernetes/delete-sensor.sh",
	}

	return renderFilenames(filenames, fields)
}
