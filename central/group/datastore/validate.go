package datastore

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// groupIDPrefix should be prepended to every human-hostile ID of a
// group for readability, e.g.,
//     "io.stackrox.authz.group.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
const groupIDPrefix = "io.stackrox.authz.group."

// ValidateGroup validates the given group for conformity.
// A group must fulfill the following:
//	- have valid properties (validated via ValidateProps).
//	- have a role name set.
func ValidateGroup(group *storage.Group) error {
	if group.GetProps() == nil {
		return errors.New("group properties must be set")
	}
	if err := ValidateProps(group.GetProps()); err != nil {
		return errors.Wrap(err, "invalid group properties")
	}
	if group.GetRoleName() == "" {
		return errors.New("groups must match to roles")
	}
	return nil
}

// ValidateProps validates the given properties for conformity.
// A property must fulfill the following:
//	- have an auth provider ID.
// 	- if no key is given, no value shall be given.
func ValidateProps(props *storage.GroupProperties) error {
	// TODO(ROX-11592): Once retrieving properties by their ID is fully deprecated, require IDs and validate this here.
	if props.GetAuthProviderId() == "" {
		return errors.Errorf("authprovider ID must be set in {%s}", proto.MarshalTextString(props))
	}
	if props.GetKey() == "" && props.GetValue() != "" {
		return errors.Errorf("cannot have a value without a key in {%s}", proto.MarshalTextString(props))
	}
	if props.GetKey() == "" && props.GetValue() == "" && props.GetTraits().GetMutabilityMode() == storage.MutabilityMode_ALLOW_FORCED {
		return errors.Errorf("default group cannot be immutable")
	}
	return nil
}

// GenerateGroupID will generate a new unique identifier for a group.
func GenerateGroupID() string {
	return groupIDPrefix + uuid.NewV4().String()
}
