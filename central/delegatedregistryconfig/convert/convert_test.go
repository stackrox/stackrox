package convert

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

var (
	storageEnabledForNone     = storage.DelegatedRegistryConfig_NONE
	storageEnabledForAll      = storage.DelegatedRegistryConfig_ALL
	storageEnabledForSpecific = storage.DelegatedRegistryConfig_SPECIFIC
	storageEnabledForInvalid  = storage.DelegatedRegistryConfig_EnabledFor(99)

	apiEnabledForNone     = v1.DelegatedRegistryConfig_NONE
	apiEnabledForAll      = v1.DelegatedRegistryConfig_ALL
	apiEnabledForSpecific = v1.DelegatedRegistryConfig_SPECIFIC
	apiEnabledForInvalid  = v1.DelegatedRegistryConfig_EnabledFor(99)

	innerAPIEnabledForNone     = central.DelegatedRegistryConfig_NONE
	innerAPIEnabledForAll      = central.DelegatedRegistryConfig_ALL
	innerAPIEnabledForSpecific = central.DelegatedRegistryConfig_SPECIFIC

	multiStorageRegs = []*storage.DelegatedRegistryConfig_DelegatedRegistry{
		{ClusterId: "id1", Path: "reg.example.com/dev"},
		{ClusterId: "id2", Path: "reg.example.com/prod"},
	}

	multiAPIRegs = []*v1.DelegatedRegistryConfig_DelegatedRegistry{
		{ClusterId: "id1", Path: "reg.example.com/dev"},
		{ClusterId: "id2", Path: "reg.example.com/prod"},
	}

	multiInnerAPIRegs = []*central.DelegatedRegistryConfig_DelegatedRegistry{
		{Path: "reg.example.com/dev"},
		{Path: "reg.example.com/prod"},
	}
)

func genStorage(enabledFor storage.DelegatedRegistryConfig_EnabledFor, defID string, regs []*storage.DelegatedRegistryConfig_DelegatedRegistry) *storage.DelegatedRegistryConfig {
	return &storage.DelegatedRegistryConfig{
		EnabledFor:       enabledFor,
		DefaultClusterId: defID,
		Registries:       regs,
	}
}
func genAPI(enabledFor v1.DelegatedRegistryConfig_EnabledFor, defID string, regs []*v1.DelegatedRegistryConfig_DelegatedRegistry) *v1.DelegatedRegistryConfig {
	return &v1.DelegatedRegistryConfig{
		EnabledFor:       enabledFor,
		DefaultClusterId: defID,
		Registries:       regs,
	}
}

func genInnerAPI(enabledFor central.DelegatedRegistryConfig_EnabledFor, regs []*central.DelegatedRegistryConfig_DelegatedRegistry) *central.DelegatedRegistryConfig {
	return &central.DelegatedRegistryConfig{
		EnabledFor: enabledFor,
		Registries: regs,
	}
}

func TestStorageToPublicAPI(t *testing.T) {
	tt := map[string]struct {
		in   *storage.DelegatedRegistryConfig
		want *v1.DelegatedRegistryConfig
	}{
		"full":            {genStorage(storageEnabledForNone, "fake", multiStorageRegs), genAPI(apiEnabledForNone, "fake", multiAPIRegs)},
		"all":             {genStorage(storageEnabledForAll, "fake", nil), genAPI(apiEnabledForAll, "fake", nil)},
		"specific":        {genStorage(storageEnabledForSpecific, "fake", nil), genAPI(apiEnabledForSpecific, "fake", nil)},
		"invalid to none": {genStorage(storageEnabledForInvalid, "fake", nil), genAPI(apiEnabledForNone, "fake", nil)},
		"nil":             {nil, nil},
	}

	for name, test := range tt {
		tf := func(t *testing.T) {
			got := StorageToPublicAPI(test.in)
			assert.Equal(t, test.want.GetEnabledFor(), got.GetEnabledFor())
			assert.Equal(t, test.want.GetDefaultClusterId(), got.GetDefaultClusterId())
			assert.Equal(t, test.want.GetRegistries(), got.GetRegistries())
		}

		t.Run(name, tf)
	}
}

func TestPublicAPIToStorage(t *testing.T) {
	tt := map[string]struct {
		in   *v1.DelegatedRegistryConfig
		want *storage.DelegatedRegistryConfig
	}{
		"full":            {genAPI(apiEnabledForNone, "fake", multiAPIRegs), genStorage(storageEnabledForNone, "fake", multiStorageRegs)},
		"all":             {genAPI(apiEnabledForAll, "fake", nil), genStorage(storageEnabledForAll, "fake", nil)},
		"specific":        {genAPI(apiEnabledForSpecific, "fake", nil), genStorage(storageEnabledForSpecific, "fake", nil)},
		"invalid to none": {genAPI(apiEnabledForInvalid, "fake", nil), genStorage(storageEnabledForNone, "fake", nil)},
		"nil":             {nil, nil},
	}

	for name, test := range tt {
		tf := func(t *testing.T) {
			got := PublicAPIToStorage(test.in)
			assert.Equal(t, test.want.GetEnabledFor(), got.GetEnabledFor())
			assert.Equal(t, test.want.GetDefaultClusterId(), got.GetDefaultClusterId())
			assert.Equal(t, test.want.GetRegistries(), got.GetRegistries())
		}

		t.Run(name, tf)
	}
}

func TestPublicAPIToInternalAPI(t *testing.T) {
	tt := map[string]struct {
		in   *v1.DelegatedRegistryConfig
		want *central.DelegatedRegistryConfig
	}{
		"full":            {genAPI(apiEnabledForNone, "fake", multiAPIRegs), genInnerAPI(innerAPIEnabledForNone, multiInnerAPIRegs)},
		"all":             {genAPI(apiEnabledForAll, "fake", nil), genInnerAPI(innerAPIEnabledForAll, nil)},
		"specific":        {genAPI(apiEnabledForSpecific, "fake", nil), genInnerAPI(innerAPIEnabledForSpecific, nil)},
		"invalid to none": {genAPI(apiEnabledForInvalid, "fake", nil), genInnerAPI(innerAPIEnabledForNone, nil)},
		"nil":             {nil, nil},
	}

	for name, test := range tt {
		tf := func(t *testing.T) {
			got := PublicAPIToInternalAPI(test.in)
			assert.Equal(t, test.want.GetEnabledFor(), got.GetEnabledFor())
			assert.Equal(t, test.want.GetRegistries(), got.GetRegistries())
		}

		t.Run(name, tf)
	}
}

func TestStorageToInternalAPI(t *testing.T) {
	tt := map[string]struct {
		in   *storage.DelegatedRegistryConfig
		want *central.DelegatedRegistryConfig
	}{
		"full":            {genStorage(storageEnabledForNone, "fake", multiStorageRegs), genInnerAPI(innerAPIEnabledForNone, multiInnerAPIRegs)},
		"all":             {genStorage(storageEnabledForAll, "fake", nil), genInnerAPI(innerAPIEnabledForAll, nil)},
		"specific":        {genStorage(storageEnabledForSpecific, "fake", nil), genInnerAPI(innerAPIEnabledForSpecific, nil)},
		"invalid to none": {genStorage(storageEnabledForInvalid, "fake", nil), genInnerAPI(innerAPIEnabledForNone, nil)},
		"nil":             {nil, nil},
	}

	for name, test := range tt {
		tf := func(t *testing.T) {
			got := StorageToInternalAPI(test.in)
			assert.Equal(t, test.want.GetEnabledFor(), got.GetEnabledFor())
			assert.Equal(t, test.want.GetRegistries(), got.GetRegistries())
		}

		t.Run(name, tf)
	}
}
