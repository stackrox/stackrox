package resources

import (
	"math"
	"testing"

	"github.com/stackrox/rox/pkg/protoconv/resources/volumes"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func TestConvertQuantityToCores(t *testing.T) {
	cases := []struct {
		quantity resource.Quantity
		expected float32
	}{
		{
			quantity: resource.MustParse("20m"),
			expected: 0.02,
		},
		{
			quantity: resource.MustParse("200m"),
			expected: 0.2,
		},
		{
			quantity: resource.MustParse("2"),
			expected: 2.0,
		},
	}

	for _, c := range cases {
		t.Run(c.quantity.String(), func(t *testing.T) {
			assert.Equal(t, c.expected, convertQuantityToCores(&c.quantity))
		})
	}
}

func TestConvertQuantityToMb(t *testing.T) {
	cases := []struct {
		quantity resource.Quantity
		expected float32
	}{
		{
			quantity: resource.MustParse("128974848"),
			expected: 123,
		},
		{
			quantity: resource.MustParse("129e6"),
			expected: 123,
		},
		{
			quantity: resource.MustParse("129M"),
			expected: 123,
		},
		{
			quantity: resource.MustParse("123Mi"),
			expected: 123,
		},
	}

	for _, c := range cases {
		t.Run(c.quantity.String(), func(t *testing.T) {
			assert.True(t, math.Abs(float64(c.expected-convertQuantityToMb(&c.quantity))) < 0.1)
		})
	}
}
