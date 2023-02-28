package user

import (
	"sort"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
	"go.uber.org/zap"
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

type loggableAuthProvider struct {
	Id   string
	Name string
	Type string
}

func protoToJSON(log *logging.Logger, message proto.Message) string {
	result, err := jsonutil.ProtoToJSON(message, jsonutil.OptCompact, jsonutil.OptUnEscape)
	if err != nil {
		log.Error("Failed to convert proto to JSON: ", err)
		return ""
	}
	return result
}

// LogSuccessfulUserLogin logs user attributes in the specified logger instance.
func LogSuccessfulUserLogin(log *logging.Logger, user *v1.AuthStatus) {
	serviceIdStr := ""
	permissionsStr := ""
	if user.GetServiceId() != nil {
		serviceIdStr = protoToJSON(log, user.GetServiceId())
	}
	if user.GetUserInfo().GetPermissions() != nil {
		permissionsStr = protoToJSON(log, user.GetUserInfo().GetPermissions())
	}
	log.Warnw("User successfully logged in with user attributes",
		zap.String("userID", user.GetUserId()),
		zap.String("serviceID", serviceIdStr),
		zap.Any("expires", user.GetExpires()),
		zap.String("username", user.GetUserInfo().GetUsername()),
		zap.String("friendlyName", user.GetUserInfo().GetFriendlyName()),
		zap.Any("roleNames", utils.RoleNamesFromUserInfo(user.GetUserInfo().GetRoles())),
		zap.String("permissions", permissionsStr),
		zap.Any("authProvider", &loggableAuthProvider{
			Id:   user.GetAuthProvider().GetId(),
			Type: user.GetAuthProvider().GetType(),
			Name: user.GetAuthProvider().GetName(),
		}))
}
