package resources

import (
	"testing"

	"github.com/stackrox/rox/pkg/protoconv/resources/volumes"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestGetVolumeSourceMap(t *testing.T) {
	t.Parallel()

	secretVol := v1.Volume{
		Name: "secret",
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: "private_key",
			},
		},
	}
	hostPathVol := v1.Volume{
		Name: "host",
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: "/var/run/docker.sock",
			},
		},
	}
	ebsVol := v1.Volume{
		Name: "ebs",
		VolumeSource: v1.VolumeSource{
			AWSElasticBlockStore: &v1.AWSElasticBlockStoreVolumeSource{
				VolumeID: "ebsVolumeID",
			},
		},
	}
	unimplementedVol := v1.Volume{
		Name: "unimplemented",
		VolumeSource: v1.VolumeSource{
			Flocker: &v1.FlockerVolumeSource{},
		},
	}

	spec := v1.PodSpec{
		Volumes: []v1.Volume{secretVol, hostPathVol, ebsVol, unimplementedVol},
	}

	expectedMap := map[string]volumes.VolumeSource{
		"secret":        volumes.VolumeRegistry["Secret"](secretVol.Secret),
		"host":          volumes.VolumeRegistry["HostPath"](hostPathVol.HostPath),
		"ebs":           volumes.VolumeRegistry["AWSElasticBlockStore"](ebsVol.AWSElasticBlockStore),
		"unimplemented": &volumes.Unimplemented{},
	}
	w := &DeploymentWrap{}
	assert.Equal(t, expectedMap, w.getVolumeSourceMap(spec))
}
