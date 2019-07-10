package clusters

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/zip"
)

func init() {
	deployers[storage.ClusterType_OPENSHIFT_CLUSTER] = newOpenshift()
}

type openshift struct{}

func newOpenshift() Deployer {
	return &openshift{}
}

func (o *openshift) Render(c Wrap, _ []byte) ([]*zip.File, error) {
	fields, err := fieldsFromWrap(c)
	if err != nil {
		return nil, err
	}
	fields["OpenshiftAPIEnv"] = env.OpenshiftAPI.EnvVar()
	fields["K8sCommand"] = "oc"

	filenames := renderer.FileNameMap{
		"kubernetes/common/delete-ca.sh": "delete-ca-sensor.sh",
		"kubernetes/common/ca-setup.sh":  "ca-setup-sensor.sh",
	}
	filenames.Add(
		"kubernetes/kubectl/sensor.yaml",
		"kubernetes/kubectl/sensor-netpol.yaml",
		"openshift/kubectl/delete-sensor.sh",
		"openshift/kubectl/sensor.sh",
		"openshift/kubectl/sensor-image-setup.sh",
		"openshift/kubectl/sensor-rbac.yaml",
	)

	if c.MonitoringEndpoint != "" {
		filenames.Add(monitoringFilenames...)
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
