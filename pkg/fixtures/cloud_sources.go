package fixtures

import (
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetStorageCloudSource returns a mock cloud source.
func GetStorageCloudSource() *storage.CloudSource {
	return &storage.CloudSource{
		Id:          "0925514f-3a33-5931-b431-756406e1a008",
		Name:        "test-integration",
		Type:        storage.CloudSource_TYPE_PALADIN_CLOUD,
		Credentials: &storage.CloudSource_Credentials{Secret: "123", ClientId: "456", ClientSecret: "789"},
		Config: &storage.CloudSource_PaladinCloud{
			PaladinCloud: &storage.PaladinCloudConfig{Endpoint: "https://apiqa.paladincloud.io"},
		},
	}
}

// GetV1CloudSource returns a mock cloud source.
func GetV1CloudSource() *v1.CloudSource {
	return &v1.CloudSource{
		Id:                  "0925514f-3a33-5931-b431-756406e1a008",
		Name:                "test-integration",
		Type:                v1.CloudSource_TYPE_PALADIN_CLOUD,
		Credentials:         &v1.CloudSource_Credentials{Secret: "123", ClientId: "456", ClientSecret: "789"},
		SkipTestIntegration: true,
		Config: &v1.CloudSource_PaladinCloud{
			PaladinCloud: &v1.PaladinCloudConfig{Endpoint: "https://apiqa.paladincloud.io"},
		},
	}
}

// GetManyStorageCloudSources returns the given number of cloud sources.
func GetManyStorageCloudSources(num int) []*storage.CloudSource {
	res := make([]*storage.CloudSource, 0, num)
	for i := 0; i < num; i++ {
		cloudSource := &storage.CloudSource{
			Id:          uuid.NewV4().String(),
			Name:        fmt.Sprintf("sample name %02d", i),
			Credentials: &storage.CloudSource_Credentials{Secret: "123", ClientId: "456", ClientSecret: "789"},
		}
		if i < num/2 {
			cloudSource.Type = storage.CloudSource_TYPE_PALADIN_CLOUD
			cloudSource.Config = &storage.CloudSource_PaladinCloud{
				PaladinCloud: &storage.PaladinCloudConfig{Endpoint: "https://apiqa.paladincloud.io"},
			}
		} else {
			cloudSource.Type = storage.CloudSource_TYPE_OCM
			cloudSource.Config = &storage.CloudSource_Ocm{
				Ocm: &storage.OCMConfig{Endpoint: "https://api.stage.openshift.com"},
			}
		}
		res = append(res, cloudSource)
	}
	return res
}
