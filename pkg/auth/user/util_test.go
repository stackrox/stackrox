package user

import (
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestExtractUserLogFields_MainFieldsTransformed(t *testing.T) {
	user := &v1.AuthStatus{
		Id: &v1.AuthStatus_UserId{
			UserId: "UserID",
		},
		AuthProvider: &storage.AuthProvider{
			Id:   "authProviderId",
			Name: "authProviderName",
			Type: "authProviderType",
		},
		Expires: protoconv.ConvertTimeToTimestampOrNil(time.Now()),
		UserAttributes: ConvertAttributes(map[string][]string{
			"a": {"b"},
		}),
		UserInfo: &storage.UserInfo{
			Username:     "DO",
			FriendlyName: "Door Opener",
			Permissions: &storage.UserInfo_ResourceToAccess{ResourceToAccess: map[string]storage.Access{
				"Open Magic Doors":  storage.Access_READ_WRITE_ACCESS,
				"Close Magic Doors": storage.Access_READ_ACCESS,
			}},
			Roles: []*storage.UserInfo_Role{
				{
					Name: "Admin",
				},
				{
					Name: "Analyst",
				},
			},
		},
	}
	fields := extractUserLogFields(user)
	assert.Len(t, fields, 9)
	assert.True(t, fields[0].(zap.Field).Equals(zap.String("userID", user.GetUserId())))
	assert.True(t, fields[1].(zap.Field).Equals(zap.String("serviceID", "")))
	assert.True(t, fields[2].(zap.Field).Equals(zap.Any("expires", user.GetExpires())))
	assert.True(t, fields[3].(zap.Field).Equals(zap.String("username", user.GetUserInfo().GetUsername())))
	assert.True(t, fields[4].(zap.Field).Equals(zap.String("friendlyName", user.GetUserInfo().GetFriendlyName())))
	assert.True(t, fields[5].(zap.Field).Equals(zap.Any("roleNames", []string{"Admin", "Analyst"})))
	assert.True(t, fields[6].(zap.Field).Equals(zap.String("permissions", "{\"resourceToAccess\":{\"Close Magic Doors\":\"READ_ACCESS\",\"Open Magic Doors\":\"READ_WRITE_ACCESS\"}}")))
	assert.True(t, fields[7].(zap.Field).Equals(zap.Any("authProvider", &loggableAuthProvider{
		ID:   user.GetAuthProvider().GetId(),
		Name: user.GetAuthProvider().GetName(),
		Type: user.GetAuthProvider().GetType(),
	})))
	assert.True(t, fields[8].(zap.Field).Equals(zap.Any("userAttributes", user.GetUserAttributes())))
}

func TestExtractUserLogFields_ServiceIdTransformed(t *testing.T) {
	user := &v1.AuthStatus{
		Id: &v1.AuthStatus_ServiceId{
			ServiceId: &storage.ServiceIdentity{
				Id:           "id",
				InitBundleId: "initBundleId",
				Type:         storage.ServiceType_CENTRAL_SERVICE,
				SerialStr:    "serialStr",
			},
		},
	}
	fields := extractUserLogFields(user)
	assert.Len(t, fields, 9)
	assert.True(t, fields[0].(zap.Field).Equals(zap.String("userID", "")))
	assert.True(t, fields[1].(zap.Field).Equals(zap.String("serviceID", "{\"serialStr\":\"serialStr\",\"id\":\"id\",\"type\":\"CENTRAL_SERVICE\",\"initBundleId\":\"initBundleId\"}")))
	assert.True(t, fields[2].(zap.Field).Equals(zap.Any("expires", user.GetExpires())))
	assert.True(t, fields[3].(zap.Field).Equals(zap.String("username", "")))
	assert.True(t, fields[4].(zap.Field).Equals(zap.String("friendlyName", "")))
	assert.True(t, fields[5].(zap.Field).Equals(zap.Any("roleNames", []string{})))
	assert.True(t, fields[6].(zap.Field).Equals(zap.String("permissions", "")))
	assert.True(t, fields[7].(zap.Field).Equals(zap.Any("authProvider", &loggableAuthProvider{
		ID:   "",
		Name: "",
		Type: "",
	})))
	assert.True(t, fields[8].(zap.Field).Equals(zap.Any("userAttributes", user.GetUserAttributes())))

}

func TestExtractUserLogFields_NilTransformed(t *testing.T) {
	var user *v1.AuthStatus
	fields := extractUserLogFields(user)
	assert.Len(t, fields, 9)
	assert.True(t, fields[0].(zap.Field).Equals(zap.String("userID", "")))
	assert.True(t, fields[1].(zap.Field).Equals(zap.String("serviceID", "")))
	assert.True(t, fields[2].(zap.Field).Equals(zap.Any("expires", user.GetExpires())))
	assert.True(t, fields[3].(zap.Field).Equals(zap.String("username", "")))
	assert.True(t, fields[4].(zap.Field).Equals(zap.String("friendlyName", "")))
	assert.True(t, fields[5].(zap.Field).Equals(zap.Any("roleNames", []string{})))
	assert.True(t, fields[6].(zap.Field).Equals(zap.String("permissions", "")))
	assert.True(t, fields[7].(zap.Field).Equals(zap.Any("authProvider", &loggableAuthProvider{
		ID:   "",
		Name: "",
		Type: "",
	})))
	assert.True(t, fields[8].(zap.Field).Equals(zap.Any("userAttributes", user.GetUserAttributes())))
}
