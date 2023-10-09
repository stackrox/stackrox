package user

import (
	"sort"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

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

type loggableAuthProvider struct {
	ID   string
	Name string
	Type string
}

func protoToJSON(message proto.Message) string {
	result, err := jsonutil.ProtoToJSON(message, jsonutil.OptCompact, jsonutil.OptUnEscape)
	if err != nil {
		log.Error("Failed to convert proto to JSON: ", err)
		return ""
	}
	return result
}

// LogSuccessfulUserLogin logs user attributes in the specified logger instance.
func LogSuccessfulUserLogin(logger logging.Logger, user *v1.AuthStatus) {
	logger.Warnw("User successfully logged in with user attributes", extractUserLogFields(user)...)
}

// The reason this function returns []interface{} instead of []zap.Field
// is because log.Warnw accepts ...interface{} and []zap.Field does not convert automatically
// to []interface{}.
func extractUserLogFields(user *v1.AuthStatus) []interface{} {
	serviceIDJSON := ""
	if user.GetServiceId() != nil {
		serviceIDJSON = protoToJSON(user.GetServiceId())
	}
	return []interface{}{
		logging.String("userID", user.GetUserId()),
		logging.String("serviceID", serviceIDJSON),
		logging.Any("expires", user.GetExpires()),
		logging.String("username", user.GetUserInfo().GetUsername()),
		logging.String("friendlyName", user.GetUserInfo().GetFriendlyName()),
		logging.Any("roleNames", utils.RoleNamesFromUserInfo(user.GetUserInfo().GetRoles())),
		logging.Any("authProvider", &loggableAuthProvider{
			ID:   user.GetAuthProvider().GetId(),
			Type: user.GetAuthProvider().GetType(),
			Name: user.GetAuthProvider().GetName(),
		}),
		logging.Any("userAttributes", user.GetUserAttributes()),
	}
}
