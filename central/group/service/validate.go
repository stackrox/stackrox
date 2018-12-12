package service

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
)

func validate(group *storage.Group) error {
	if err := validateProps(group.GetProps()); err != nil {
		return err
	}
	if group.GetRoleName() == "" {
		return fmt.Errorf("groups must match to roles")
	}
	return nil
}

func validateProps(props *storage.GroupProperties) error {
	if props.GetKey() == "" && props.GetValue() != "" {
		return fmt.Errorf("cannot have a value without a key in group properties: %s", proto.MarshalTextString(props))
	}
	return nil
}
