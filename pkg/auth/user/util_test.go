package user

import (
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestExtractUserLogFields_MainFieldsTransformed(t *testing.T) {
	user := v1.AuthStatus_builder{
		UserId: proto.String("UserID"),
		AuthProvider: storage.AuthProvider_builder{
			Id:   "authProviderId",
			Name: "authProviderName",
			Type: "authProviderType",
		}.Build(),
		Expires: protoconv.ConvertTimeToTimestampOrNil(time.Now()),
		UserAttributes: ConvertAttributes(map[string][]string{
			"a": {"b"},
		}),
		UserInfo: storage.UserInfo_builder{
			Username:     "DO",
			FriendlyName: "Door Opener",
			Permissions: storage.UserInfo_ResourceToAccess_builder{ResourceToAccess: map[string]storage.Access{
				"Open Magic Doors":  storage.Access_READ_WRITE_ACCESS,
				"Close Magic Doors": storage.Access_READ_ACCESS,
			}}.Build(),
			Roles: []*storage.UserInfo_Role{
				storage.UserInfo_Role_builder{
					Name: "Admin",
				}.Build(),
				storage.UserInfo_Role_builder{
					Name: "Analyst",
				}.Build(),
			},
		}.Build(),
	}.Build()
	fields := extractUserLogFields(user)
	assert.Len(t, fields, 8)
	assert.Contains(t, fields, logging.String("userID", user.GetUserId()))
	assert.Contains(t, fields, logging.String("serviceID", ""))
	assert.Contains(t, fields, logging.Any("expires", user.GetExpires()))
	assert.Contains(t, fields, logging.String("username", user.GetUserInfo().GetUsername()))
	assert.Contains(t, fields, logging.String("friendlyName", user.GetUserInfo().GetFriendlyName()))
	assert.Contains(t, fields, logging.Any("roleNames", []string{"Admin", "Analyst"}))
	assert.Contains(t, fields, logging.Any("authProvider", &loggableAuthProvider{
		ID:   user.GetAuthProvider().GetId(),
		Name: user.GetAuthProvider().GetName(),
		Type: user.GetAuthProvider().GetType(),
	}))
	assert.Contains(t, fields, logging.Any("userAttributes", user.GetUserAttributes()))
}

func TestExtractUserLogFields_ServiceIdTransformed(t *testing.T) {
	si := &storage.ServiceIdentity{}
	si.SetId("id")
	si.SetInitBundleId("initBundleId")
	si.SetType(storage.ServiceType_CENTRAL_SERVICE)
	si.SetSerialStr("serialStr")
	user := &v1.AuthStatus{}
	user.SetServiceId(proto.ValueOrDefault(si))
	fields := extractUserLogFields(user)
	assert.Len(t, fields, 8)
	assert.Contains(t, fields, logging.String("userID", ""))
	assert.Contains(t, fields, logging.String("serviceID", protoToJSON(user.GetServiceId())))
	assert.Contains(t, fields, logging.Any("expires", user.GetExpires()))
	assert.Contains(t, fields, logging.String("username", ""))
	assert.Contains(t, fields, logging.String("friendlyName", ""))
	assert.Contains(t, fields, logging.Any("roleNames", []string{}))
	assert.Contains(t, fields, logging.Any("authProvider", &loggableAuthProvider{
		ID:   "",
		Name: "",
		Type: "",
	}))
	assert.Contains(t, fields, logging.Any("userAttributes", user.GetUserAttributes()))

}

func TestExtractUserLogFields_NilTransformed(t *testing.T) {
	var user *v1.AuthStatus
	fields := extractUserLogFields(user)
	assert.Len(t, fields, 8)
	assert.Contains(t, fields, logging.String("userID", ""))
	assert.Contains(t, fields, logging.String("serviceID", ""))
	assert.Contains(t, fields, logging.Any("expires", user.GetExpires()))
	assert.Contains(t, fields, logging.String("username", ""))
	assert.Contains(t, fields, logging.String("friendlyName", ""))
	assert.Contains(t, fields, logging.Any("roleNames", []string{}))
	assert.Contains(t, fields, logging.Any("authProvider", &loggableAuthProvider{
		ID:   "",
		Name: "",
		Type: "",
	}))
	assert.Contains(t, fields, logging.Any("userAttributes", user.GetUserAttributes()))
}

func TestProtoToJSONServiceIdentity(t *testing.T) {
	_ = t
	const svcIdentityID = "ecabcdef-bbbb-4011-0000-111111111111"
	const initBundleID = "ebaaaaaa-bbbb-4011-0000-111111111111"
	const serialString = "12345678901"
	const svcIdentityType = storage.ServiceType_CENTRAL_SERVICE
	testServiceIdentity := &storage.ServiceIdentity{}
	testServiceIdentity.SetSerialStr(serialString)
	testServiceIdentity.SetSerial(int64(12345678901))
	testServiceIdentity.SetId(svcIdentityID)
	testServiceIdentity.SetType(svcIdentityType)
	testServiceIdentity.SetInitBundleId(initBundleID)
	serialized := protoToJSON(testServiceIdentity)
	expectedSerialized := `{` +
		`"serialStr":"` + serialString + `",` +
		`"serial":"` + serialString + `",` +
		`"id":"` + svcIdentityID + `",` +
		`"type":"` + storage.ServiceType_name[int32(svcIdentityType)] + `",` +
		`"initBundleId":"` + initBundleID + `"` +
		`}`
	assert.JSONEq(t, expectedSerialized, serialized)
	assert.Len(t, strings.Split(serialized, "\n"), 1)
	// The compact form should not contain any whitespace around JSON tokens
	// (e.g. '{', '"', ':', ',', '}')
	assert.NotRegexp(t, "[:{,}\"]\\s", serialized)
	assert.NotRegexp(t, "\\s[:{,}\"]", serialized)
}
