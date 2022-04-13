package service

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
)

func validate(group *storage.Group) error {
	if group.GetProps() == nil {
		return errors.New("group properties must be set")
	}
	if err := validateProps(group.GetProps()); err != nil {
		return errors.Wrap(err, "invalid group properties")
	}
	if group.GetRoleName() == "" {
		return errors.New("groups must match to roles")
	}
	return nil
}

func validateProps(props *storage.GroupProperties) error {
	if props.GetAuthProviderId() == "" {
		return fmt.Errorf("authprovider ID must be set in {%s}", proto.MarshalTextString(props))
	}
	if props.GetKey() == "" && props.GetValue() != "" {
		return fmt.Errorf("cannot have a value without a key in {%s}", proto.MarshalTextString(props))
	}
	return nil
}
