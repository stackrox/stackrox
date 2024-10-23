package telemeter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_With(t *testing.T) {
	opts := ApplyOptions([]Option{
		WithUserID("userID"),
		WithClient("clientID", "clientType", "clientVersion"),
		WithGroups("groupA", "groupA_id1"),
		WithGroups("groupA", "groupA_id2"),
		WithGroups("groupB", "groupB_id"),
		WithNoDuplicates("test"),
	},
	)
	assert.Equal(t, "userID", opts.UserID)
	assert.Equal(t, "clientID", opts.ClientID)
	assert.Equal(t, "clientType", opts.ClientType)
	assert.Equal(t, "clientVersion", opts.ClientVersion)
	assert.Equal(t, "test", opts.MessageIDPrefix)

	props := map[string][]string{
		"groupA": {"groupA_id1", "groupA_id2"},
		"groupB": {"groupB_id"},
	}
	assert.Equal(t, props, opts.Groups)
	assert.NotNil(t, ApplyOptions(nil))
}
