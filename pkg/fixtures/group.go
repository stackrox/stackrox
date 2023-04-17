package fixtures

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

var idCounter int

// GetGroup return a mock storage.Group with all possible properties filled out.
func GetGroup() *storage.Group {
	idCounter++
	return &storage.Group{
		Props: &storage.GroupProperties{
			Id:             fmt.Sprintf("abcdef-%d", idCounter),
			AuthProviderId: "authProviderA",
			Key:            "AttributeA",
			Value:          "ValueUno",
		},
		RoleName: "test-role",
	}
}

// GetGroupWithMutability returns a mock storage.Group with all possible properties filled out.
func GetGroupWithMutability(mode storage.Traits_MutabilityMode) *storage.Group {
	group := GetGroup()

	group.Props.Traits = &storage.Traits{MutabilityMode: mode}

	return group
}

// GetGroupWithOrigin returns a mock storage.Group with all possible properties filled out and with the specified origin set.
func GetGroupWithOrigin(origin storage.Traits_Origin) *storage.Group {
	group := GetGroup()

	group.Props.Traits = &storage.Traits{Origin: origin}

	return group
}

// GetGroups returns a set of mock storage.Group objects, which in total represents the possible combinations of group
// properties and roles.
func GetGroups() []*storage.Group {
	return []*storage.Group{
		{
			Props: &storage.GroupProperties{
				Id: "0",
			},
			RoleName: "role1",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Id:             "1",
			},
			RoleName: "role2",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Id:             "2",
			},
			RoleName: "role3",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Value:          "Value1",
				Id:             "3",
			},
			RoleName: "role4",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute2",
				Value:          "Value1",
				Id:             "4",
			},
			RoleName: "role5",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvide2",
				Key:            "Attribute1",
				Value:          "Value1",
				Id:             "5",
			},
			RoleName: "role6",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvide2",
				Key:            "Attribute2",
				Value:          "Value1",
				Id:             "6",
			},
			RoleName: "role7",
		},
	}
}
