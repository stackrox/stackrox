package matcher

import (
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"go.uber.org/mock/gomock"
)

func GetTestPlatformMatcherWithDefaultPlatformComponentConfig(mockCtrl *gomock.Controller) PlatformMatcher {
	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(mockCtrl)
	mockConfigDatastore.EXPECT().GetPlatformComponentConfig(gomock.Any()).Return(GetDefaultPlatformComponentConfig(), true, nil).Times(1)

	return New(mockConfigDatastore)
}

func GetDefaultPlatformComponentConfig() *storage.PlatformComponentConfig {
	return storage.PlatformComponentConfig_builder{
		NeedsReevaluation: false,
		Rules: []*storage.PlatformComponentConfig_Rule{
			storage.PlatformComponentConfig_Rule_builder{
				Name: "system rule",
				NamespaceRule: storage.PlatformComponentConfig_Rule_NamespaceRule_builder{
					Regex: `^kube-.*|^openshift-.*`,
				}.Build(),
			}.Build(),
			storage.PlatformComponentConfig_Rule_builder{
				Name: "red hat layered products",
				NamespaceRule: storage.PlatformComponentConfig_Rule_NamespaceRule_builder{
					Regex: `^stackrox$|^rhacs-operator$|^open-cluster-management$|^multicluster-engine$|^aap$|^hive$`,
				}.Build(),
			}.Build(),
		},
	}.Build()
}
