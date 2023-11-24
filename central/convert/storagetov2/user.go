package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func convertUser(user *storage.SlimUser) *v2.SlimUser {
	if user == nil {
		return nil
	}

	return &v2.SlimUser{
		Id:   user.GetId(),
		Name: user.GetName(),
	}
}
