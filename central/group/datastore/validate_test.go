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
			group: &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: "Sentinel",
					Key:            "le mot",
					Value:          "tous a la Bastille!",
					Id:             "id",
				},
				RoleName: "",
			},
			wantErr: true,
		},
		{
			name: "Group.props must be set",
			group: &storage.Group{
				Props:    nil,
				RoleName: "insurge",
			},
			wantErr: true,
		},
		{
			name: "Group.props.id must be set",
			group: &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: "Sentinel",
					Key:            "le mot",
					Value:          "tous a la Bastille!",
				},
				RoleName: "insurge",
			},
			wantErr: true,
		},
		{
			name: "Group.props.auth_provider_id must be set",
			group: &storage.Group{
				Props: &storage.GroupProperties{
					Key:   "le mot",
					Value: "tous a la Bastille!",
					Id:    "id",
				},
				RoleName: "insurge",
			},
			wantErr: true,
		},
		{
			name: "Group.props specify value but no key",
			group: &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: "Sentinel",
					Value:          "tous a la Bastille!",
					Id:             "id",
				},
				RoleName: "insurge",
			},
			wantErr: true,
		},
		{
			name: "Basic case: auth provider maps to a role",
			group: &storage.Group{
				Props: &storage.GroupProperties{
					Id:             "1",
					AuthProviderId: "Sentinel",
				},
				RoleName: "insurge",
			},
			wantErr: false,
		},
		{
			name: "Key exists case: auth provider with a key maps to a role",
			group: &storage.Group{
				Props: &storage.GroupProperties{
					Id:             "1",
					AuthProviderId: "Sentinel",
					Key:            "le mot",
				},
				RoleName: "insurge",
			},
			wantErr: false,
		},
		{
			name: "Key/value case: auth provider with a key and a value maps to a role",
			group: &storage.Group{
				Props: &storage.GroupProperties{
					Id:             "1",
					AuthProviderId: "Sentinel",
					Key:            "le mot",
					Value:          "tous a la Bastille!",
				},
				RoleName: "insurge",
			},
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
