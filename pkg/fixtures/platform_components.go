package fixtures

import (
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	"github.com/stackrox/rox/generated/storage"
	"go.uber.org/mock/gomock"
)

func GetPlatformMatcherWithDefaultPlatformComponentConfig(mockCtrl *gomock.Controller) platformmatcher.PlatformMatcher {
	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(mockCtrl)
	mockConfigDatastore.EXPECT().GetPlatformComponentConfig(gomock.Any()).Return(GetDefaultPlatformComponentConfig(), true, nil).Times(1)

	return platformmatcher.New(mockConfigDatastore)
}

func GetDefaultPlatformComponentConfig() *storage.PlatformComponentConfig {
	return &storage.PlatformComponentConfig{
		NeedsReevaluation: false,
		Rules: []*storage.PlatformComponentConfig_Rule{
			{
				Name: "system rule",
				NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
					Regex: `^kube-.*|^openshift-.*`,
				},
			},
			{
				Name: "red hat layered products",
				NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
					Regex: `^stackrox$|^rhacs-operator$|^open-cluster-management$|^multicluster-engine$|^aap$|^hive$`,
				},
			},
		},
	}
}
