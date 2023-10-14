package service

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestConvertToV1Proto(t *testing.T) {
	testCases := []*storage.AuthMachineToMachineConfig{
		{
			Id:                      "some-id",
			Type:                    storage.AuthMachineToMachineConfig_GITHUB_ACTIONS,
			TokenExpirationDuration: "1h",
			Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
				{
					Key:   "sub",
					Value: "some-value",
					Role:  "My Special Role",
				},
			},
			IssuerConfig: &storage.AuthMachineToMachineConfig_Generic{
				Generic: &storage.AuthMachineToMachineConfig_GenericIssuer{
					Issuer: "https://stackrox.io",
				},
			},
		},
		{
			Id:                      "some-id",
			Type:                    storage.AuthMachineToMachineConfig_GITHUB_ACTIONS,
			TokenExpirationDuration: "1h",
			Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
				{
					Key:   "sub",
					Value: "some-value",
					Role:  "My Special Role",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			v1Config := toV1Proto(tc)
			assert.Equal(t, tc.GetId(), v1Config.GetId())
			assert.Equal(t, tc.GetType().String(), v1Config.GetType().String())
			assert.Equal(t, tc.GetTokenExpirationDuration(), v1Config.GetTokenExpirationDuration())
			for i, mapping := range tc.GetMappings() {
				assert.Equal(t, mapping.GetKey(), v1Config.GetMappings()[i].GetKey())
				assert.Equal(t, mapping.GetValue(), v1Config.GetMappings()[i].GetValue())
				assert.Equal(t, mapping.GetRole(), v1Config.GetMappings()[i].GetRole())
			}
			if tc.GetGeneric() != nil {
				assert.Equal(t, tc.GetGeneric().GetIssuer(), v1Config.GetGeneric().GetIssuer())
			} else {
				assert.Nil(t, v1Config.GetIssuerConfig())
				assert.Nil(t, v1Config.GetGeneric())
			}
		})
	}
}

func TestConvertToStorageProto(t *testing.T) {
	testCases := []*v1.AuthMachineToMachineConfig{
		{
			Id:                      "some-id",
			Type:                    v1.AuthMachineToMachineConfig_GITHUB_ACTIONS,
			TokenExpirationDuration: "1h",
			Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
				{
					Key:   "sub",
					Value: "some-value",
					Role:  "My special role",
				},
			},
		},
		{
			Id:                      "some-id",
			Type:                    v1.AuthMachineToMachineConfig_GENERIC,
			TokenExpirationDuration: "1h",
			Mappings: []*v1.AuthMachineToMachineConfig_Mapping{
				{
					Key:   "sub",
					Value: "some-value",
					Role:  "My special role",
				},
			},
			IssuerConfig: &v1.AuthMachineToMachineConfig_Generic{
				Generic: &v1.AuthMachineToMachineConfig_GenericIssuer{
					Issuer: "https://stackrox.io",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("tc %d", i), func(t *testing.T) {
			storageProto := toStorageProto(tc)
			assert.Equal(t, tc.GetId(), storageProto.GetId())
			assert.Equal(t, tc.GetType().String(), storageProto.GetType().String())
			assert.Equal(t, tc.GetTokenExpirationDuration(), storageProto.GetTokenExpirationDuration())
			for i, mapping := range tc.GetMappings() {
				assert.Equal(t, mapping.GetKey(), storageProto.GetMappings()[i].GetKey())
				assert.Equal(t, mapping.GetValue(), storageProto.GetMappings()[i].GetValue())
				assert.Equal(t, mapping.GetRole(), storageProto.GetMappings()[i].GetRole())
			}
			if tc.GetGeneric() != nil {
				assert.Equal(t, tc.GetGeneric().GetIssuer(), storageProto.GetGeneric().GetIssuer())
			} else {
				assert.Nil(t, storageProto.GetIssuerConfig())
				assert.Nil(t, storageProto.GetGeneric())
			}
		})
	}
}
