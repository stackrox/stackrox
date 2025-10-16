package datastore

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		group   *storage.Group
		wantErr bool
	}{
		{
			name: "Group.role_name must be set",
			group: storage.Group_builder{
				Props: storage.GroupProperties_builder{
					AuthProviderId: "Sentinel",
					Key:            "le mot",
					Value:          "tous a la Bastille!",
					Id:             "id",
				}.Build(),
				RoleName: "",
			}.Build(),
			wantErr: true,
		},
		{
			name: "Group.props must be set",
			group: storage.Group_builder{
				Props:    nil,
				RoleName: "insurge",
			}.Build(),
			wantErr: true,
		},
		{
			name: "Group.props.id must be set",
			group: storage.Group_builder{
				Props: storage.GroupProperties_builder{
					AuthProviderId: "Sentinel",
					Key:            "le mot",
					Value:          "tous a la Bastille!",
				}.Build(),
				RoleName: "insurge",
			}.Build(),
			wantErr: true,
		},
		{
			name: "Group.props.auth_provider_id must be set",
			group: storage.Group_builder{
				Props: storage.GroupProperties_builder{
					Key:   "le mot",
					Value: "tous a la Bastille!",
					Id:    "id",
				}.Build(),
				RoleName: "insurge",
			}.Build(),
			wantErr: true,
		},
		{
			name: "Group.props specify value but no key",
			group: storage.Group_builder{
				Props: storage.GroupProperties_builder{
					AuthProviderId: "Sentinel",
					Value:          "tous a la Bastille!",
					Id:             "id",
				}.Build(),
				RoleName: "insurge",
			}.Build(),
			wantErr: true,
		},
		{
			name: "Basic case: auth provider maps to a role",
			group: storage.Group_builder{
				Props: storage.GroupProperties_builder{
					Id:             "1",
					AuthProviderId: "Sentinel",
				}.Build(),
				RoleName: "insurge",
			}.Build(),
			wantErr: false,
		},
		{
			name: "Key exists case: auth provider with a key maps to a role",
			group: storage.Group_builder{
				Props: storage.GroupProperties_builder{
					Id:             "1",
					AuthProviderId: "Sentinel",
					Key:            "le mot",
				}.Build(),
				RoleName: "insurge",
			}.Build(),
			wantErr: false,
		},
		{
			name: "Key/value case: auth provider with a key and a value maps to a role",
			group: storage.Group_builder{
				Props: storage.GroupProperties_builder{
					Id:             "1",
					AuthProviderId: "Sentinel",
					Key:            "le mot",
					Value:          "tous a la Bastille!",
				}.Build(),
				RoleName: "insurge",
			}.Build(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateGroup(tt.group, true); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
