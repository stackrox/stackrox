// +build integration

package tenable

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	accessKey = "54d75bd30474079b62761b5913917c27a8bb8f781b823c2d8d51dda687180bf3"
	secretKey = "0dbf0fe9bf34117ca49b40cf36eab72c9e2cb2247739dcbd2706fdf9cc4cb0e3"
)

func TestTenable(t *testing.T) {
	protoImageIntegration := &v1.ImageIntegration{
		IntegrationConfig: &v1.ImageIntegration_Tenable{
			Tenable: &v1.TenableConfig{
				AccessKey: accessKey,
				SecretKey: secretKey,
			},
		},
	}
	reg, err := newRegistry(protoImageIntegration)
	require.NoError(t, err)

	i := &v1.Image{
		Name: &v1.ImageName{
			Remote: "srox/nginx",
			Tag:    "1.10",
		},
	}

	metadata, err := reg.Metadata(i)
	require.NoError(t, err)
	assert.Nil(t, metadata)
	assert.Equal(t, "0346349a1a640da9535acfc0f68be9d9b81e85957725ecb76f3b522f4e2f0455", i.Name.GetSha())

	err = reg.Test()
	assert.NoError(t, err)
}
