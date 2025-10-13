package v2tostorage

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func convertUsers(users []*v2.SlimUser) []*storage.SlimUser {
	if len(users) == 0 {
		return nil
	}

	var ret []*storage.SlimUser
	for _, user := range users {
		if user == nil {
			continue
		}
		ret = append(ret, convertUser(user))
	}

	return ret
}

func convertUser(user *v2.SlimUser) *storage.SlimUser {
	if user == nil {
		return nil
	}

	return &storage.SlimUser{
		Id:   user.GetId(),
		Name: user.GetName(),
	}
}
