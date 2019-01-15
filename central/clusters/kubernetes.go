package clusters

import (
	"encoding/base64"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/zip"
)

func init() {
	deployers[storage.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
}

type kubernetes struct{}

func newKubernetes() Deployer {
	return &kubernetes{}
}

func addCommonKubernetesParams(params *storage.CommonKubernetesParams, fields map[string]interface{}) {
	fields["Namespace"] = params.GetNamespace()
	fields["NamespaceEnv"] = env.Namespace.EnvVar()
}

var monitoringFilenames = []string{
	"kubernetes/kubectl/telegraf.conf",
}

var admissionController = "kubernetes/kubectl/admission-controller.yaml"

func (k *kubernetes) Render(c Wrap, ca []byte) ([]*zip.File, error) {
	var kubernetesParams *storage.KubernetesParams
	clusterKube, ok := c.OrchestratorParams.(*storage.Cluster_Kubernetes)
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

	if c.AdmissionController {
		fields["CABundle"] = base64.StdEncoding.EncodeToString(ca)
		filenames = append(filenames, admissionController)
	}

	return renderFilenames(filenames, fields, "/data/assets/docker-auth.sh")
}
