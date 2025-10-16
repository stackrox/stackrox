package fixtures

import (
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/proto"
)

// GetStorageCloudSource returns a mock cloud source.
func GetStorageCloudSource() *storage.CloudSource {
	cc := &storage.CloudSource_Credentials{}
	cc.SetSecret("123")
	cc.SetClientId("456")
	cc.SetClientSecret("789")
	pcc := &storage.PaladinCloudConfig{}
	pcc.SetEndpoint("https://apiqa.paladincloud.io")
	cs := &storage.CloudSource{}
	cs.SetId("0925514f-3a33-5931-b431-756406e1a008")
	cs.SetName("test-integration")
	cs.SetType(storage.CloudSource_TYPE_PALADIN_CLOUD)
	cs.SetCredentials(cc)
	cs.SetPaladinCloud(proto.ValueOrDefault(pcc))
	return cs
}

// GetV1CloudSource returns a mock cloud source.
func GetV1CloudSource() *v1.CloudSource {
	cc := &v1.CloudSource_Credentials{}
	cc.SetSecret("123")
	cc.SetClientId("456")
	cc.SetClientSecret("789")
	pcc := &v1.PaladinCloudConfig{}
	pcc.SetEndpoint("https://apiqa.paladincloud.io")
	cs := &v1.CloudSource{}
	cs.SetId("0925514f-3a33-5931-b431-756406e1a008")
	cs.SetName("test-integration")
	cs.SetType(v1.CloudSource_TYPE_PALADIN_CLOUD)
	cs.SetCredentials(cc)
	cs.SetSkipTestIntegration(true)
	cs.SetPaladinCloud(proto.ValueOrDefault(pcc))
	return cs
}

// GetManyStorageCloudSources returns the given number of cloud sources.
func GetManyStorageCloudSources(num int) []*storage.CloudSource {
	res := make([]*storage.CloudSource, 0, num)
	for i := 0; i < num; i++ {
		cc := &storage.CloudSource_Credentials{}
		cc.SetSecret("123")
		cc.SetClientId("456")
		cc.SetClientSecret("789")
		cloudSource := &storage.CloudSource{}
		cloudSource.SetId(uuid.NewV4().String())
		cloudSource.SetName(fmt.Sprintf("sample name %02d", i))
		cloudSource.SetCredentials(cc)
		if i < num/2 {
			cloudSource.SetType(storage.CloudSource_TYPE_PALADIN_CLOUD)
			pcc := &storage.PaladinCloudConfig{}
			pcc.SetEndpoint("https://apiqa.paladincloud.io")
			cloudSource.SetPaladinCloud(proto.ValueOrDefault(pcc))
		} else {
			cloudSource.SetType(storage.CloudSource_TYPE_OCM)
			oCMConfig := &storage.OCMConfig{}
			oCMConfig.SetEndpoint("https://api.stage.openshift.com")
			cloudSource.SetOcm(proto.ValueOrDefault(oCMConfig))
		}
		res = append(res, cloudSource)
	}
	return res
}
