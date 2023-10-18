package storagetov2

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func convertUsers(users []*storage.SlimUser) []*v2.SlimUser {
	if len(users) == 0 {
		return nil
	}

	var ret []*v2.SlimUser
	for _, user := range users {
		if user == nil {
			continue
		}
		ret = append(ret, convertUser(user))
	}

	return ret
}

func convertUser(user *storage.SlimUser) *v2.SlimUser {
	if user == nil {
		return nil
	}

	return &v2.SlimUser{
		Id:   user.GetId(),
		Name: user.GetName(),
	}
}
