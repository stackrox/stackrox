package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func convertUser(user *storage.SlimUser) *v2.SlimUser {
	if user == nil {
		return nil
	}

	slimUser := &v2.SlimUser{}
	slimUser.SetId(user.GetId())
	slimUser.SetName(user.GetName())
	return slimUser
}
