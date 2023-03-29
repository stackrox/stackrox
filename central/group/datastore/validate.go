package datastore

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// groupIDPrefix should be prepended to every human-hostile ID of a
// group for readability, e.g.,
//
//	"io.stackrox.authz.group.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
const groupIDPrefix = "io.stackrox.authz.group."

// ValidateGroup validates the given group for conformity.
// A group must fulfill the following:
//   - have valid properties (validated via ValidateProps).
//   - have a role name set.
func ValidateGroup(group *storage.Group, requireID bool) error {
	if group.GetProps() == nil {
		return errors.New("group properties must be set")
	}
	if err := ValidateProps(group.GetProps(), requireID); err != nil {
		return errors.Wrap(err, "invalid group properties")
	}
	if group.GetRoleName() == "" {
		return errors.New("groups must match to roles")
	}
	return nil
}

// ValidateProps validates the given properties for conformity.
// A property must fulfill the following:
//   - have an auth provider ID.
//   - if no key is given, no value shall be given.
func ValidateProps(props *storage.GroupProperties, requireID bool) error {
	if requireID && props.GetId() == "" {
		return errors.Errorf("group ID must be set in {%s}", proto.MarshalTextString(props))
	}
	if props.GetAuthProviderId() == "" {
		return errors.Errorf("authprovider ID must be set in {%s}", proto.MarshalTextString(props))
	}
	if props.GetKey() == "" && props.GetValue() != "" {
		return errors.Errorf("cannot have a value without a key in {%s}", proto.MarshalTextString(props))
	}
	if props.GetKey() == "" && props.GetValue() == "" &&
		props.GetTraits().GetMutabilityMode() == storage.Traits_ALLOW_MUTATE_FORCED {
		return errors.New("default group cannot be immutable")
	}
	return nil
}

// GenerateGroupID will generate a new unique identifier for a group.
func GenerateGroupID() string {
	return groupIDPrefix + uuid.NewV4().String()
}
