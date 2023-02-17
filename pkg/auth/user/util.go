package user

import (
	"sort"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
)

// ConvertAttributes converts a map of user attributes to v1.UserAttribute
func ConvertAttributes(attrMap map[string][]string) []*v1.UserAttribute {
	if attrMap == nil {
		return nil
	}

	result := make([]*v1.UserAttribute, 0, len(attrMap))
	for k, vs := range attrMap {
		attr := &v1.UserAttribute{
			Key:    k,
			Values: vs,
		}
		result = append(result, attr)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})
	return result
}

// LogSuccessfulUserLogin logs user attributes in the specified logger instance.
func LogSuccessfulUserLogin(log *logging.Logger, user *v1.AuthStatus) {
	loggableUser := user.Clone()
	// Auth provider config can contain sensitive data(client secret, certificates etc.) so it shouldn't be logged.
	if loggableUser != nil {
		loggableUser.AuthProvider = nil
	}
	userJSON, _ := jsonutil.ProtoToJSON(loggableUser)
	log.Warnf("User successfully logged in with user attributes: %s", userJSON)
}
