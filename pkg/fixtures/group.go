package fixtures

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

var idCounter int

// GetGroup return a mock storage.Group with all possible properties filled out.
func GetGroup() *storage.Group {
	idCounter++
	gp := &storage.GroupProperties{}
	gp.SetId(fmt.Sprintf("abcdef-%d", idCounter))
	gp.SetAuthProviderId("authProviderA")
	gp.SetKey("AttributeA")
	gp.SetValue("ValueUno")
	group := &storage.Group{}
	group.SetProps(gp)
	group.SetRoleName("test-role")
	return group
}

// GetGroupWithMutability returns a mock storage.Group with all possible properties filled out.
func GetGroupWithMutability(mode storage.Traits_MutabilityMode) *storage.Group {
	group := GetGroup()

	traits := &storage.Traits{}
	traits.SetMutabilityMode(mode)
	group.GetProps().SetTraits(traits)

	return group
}

// GetGroupWithOrigin returns a mock storage.Group with all possible properties filled out and with the specified origin set.
func GetGroupWithOrigin(origin storage.Traits_Origin) *storage.Group {
	group := GetGroup()

	traits := &storage.Traits{}
	traits.SetOrigin(origin)
	group.GetProps().SetTraits(traits)

	return group
}

// GetGroups returns a set of mock storage.Group objects, which in total represents the possible combinations of group
// properties and roles.
func GetGroups() []*storage.Group {
	return []*storage.Group{
		storage.Group_builder{
			Props: storage.GroupProperties_builder{
				Id: "0",
			}.Build(),
			RoleName: "role1",
		}.Build(),
		storage.Group_builder{
			Props: storage.GroupProperties_builder{
				AuthProviderId: "authProvider1",
				Id:             "1",
			}.Build(),
			RoleName: "role2",
		}.Build(),
		storage.Group_builder{
			Props: storage.GroupProperties_builder{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Id:             "2",
			}.Build(),
			RoleName: "role3",
		}.Build(),
		storage.Group_builder{
			Props: storage.GroupProperties_builder{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Value:          "Value1",
				Id:             "3",
			}.Build(),
			RoleName: "role4",
		}.Build(),
		storage.Group_builder{
			Props: storage.GroupProperties_builder{
				AuthProviderId: "authProvider1",
				Key:            "Attribute2",
				Value:          "Value1",
				Id:             "4",
			}.Build(),
			RoleName: "role5",
		}.Build(),
		storage.Group_builder{
			Props: storage.GroupProperties_builder{
				AuthProviderId: "authProvide2",
				Key:            "Attribute1",
				Value:          "Value1",
				Id:             "5",
			}.Build(),
			RoleName: "role6",
		}.Build(),
		storage.Group_builder{
			Props: storage.GroupProperties_builder{
				AuthProviderId: "authProvide2",
				Key:            "Attribute2",
				Value:          "Value1",
				Id:             "6",
			}.Build(),
			RoleName: "role7",
		}.Build(),
	}
}
