package clusters

import (
	"encoding/base64"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/admissioncontrol"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/zip"
)

func init() {
	deployers[storage.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
}

type kubernetes struct{}

func newKubernetes() Deployer {
	return &kubernetes{}
}

var admissionController = "kubernetes/kubectl/admission-controller.yaml"

func (*kubernetes) Render(cluster *storage.Cluster, ca []byte, opts RenderOptions) ([]*zip.File, error) {
	fields, err := fieldsFromClusterAndRenderOpts(cluster, opts)
	if err != nil {
		return nil, err
	}

	fields["K8sCommand"] = "kubectl"

	filenames := renderer.FileNameMap{
		"kubernetes/common/ca-setup.sh":  "ca-setup-sensor.sh",
		"kubernetes/common/delete-ca.sh": "delete-ca-sensor.sh",
	}
	filenames.Add(
		"kubernetes/kubectl/sensor.sh",
		"kubernetes/kubectl/sensor.yaml",
		"kubernetes/kubectl/sensor-rbac.yaml",
		"kubernetes/kubectl/sensor-netpol.yaml",
		"kubernetes/kubectl/delete-sensor.sh",
		"kubernetes/kubectl/sensor-pod-security.yaml",
		"kubernetes/kubectl/upgrader-serviceaccount.yaml",
	)

	if cluster.AdmissionController {
		fields["CABundle"] = base64.StdEncoding.EncodeToString(ca)
		fields["AdmissionControlService"] = features.AdmissionControlService.Enabled()
		if features.AdmissionControlService.Enabled() {
			fields["AdmissionControlListenOnUpdates"] = features.AdmissionControlEnforceOnUpdate.Enabled() && cluster.GetAdmissionControllerUpdates()
			fields["AdmissionControlConfigMapName"] = admissioncontrol.ConfigMapName
		}

		filenames.Add(admissionController)
	}

	allFiles, err := renderer.RenderFiles(filenames, fields)
	if err != nil {
		return nil, err
	}

	assetFiles, err := renderer.LoadAssets(renderer.NewFileNameMap(dockerAuthAssetFile))
	if err != nil {
		return nil, err
	}

	allFiles = append(allFiles, assetFiles...)
	return allFiles, err
}
