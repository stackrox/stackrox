package segment

import (
	"testing"

	segment "github.com/segmentio/analytics-go/v3"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getMessageType(t *testing.T) {
	track := segment.Track{
		Type: "Track",
	}

	assert.Equal(t, "Track", getMessageType(track))
}

func Test_makeDeviceContext(t *testing.T) {
	opts := telemeter.ApplyOptions([]telemeter.Option{
		telemeter.WithUserID("userID"),
		telemeter.WithClient("clientID", "clientType"),
		telemeter.WithGroups("groupA", "groupA_id1"),
		telemeter.WithGroups("groupA", "groupA_id2"),
		telemeter.WithGroups("groupB", "groupB_id"),
	})

	ctx := makeDeviceContext(opts)
	assert.Equal(t, "clientID", ctx.Device.Id)
	assert.Equal(t, "clientType", ctx.Device.Type)

	require.Contains(t, ctx.Extra, "groups")
	require.Contains(t, ctx.Extra["groups"], "groupA")
	assert.Contains(t, ctx.Extra["groups"], "groupB")
	groups := ctx.Extra["groups"].(map[string][]string)
	assert.ElementsMatch(t, []string{"groupA_id1", "groupA_id2"}, groups["groupA"])
}

func Test_getIDs(t *testing.T) {
	type result struct {
		anonymousID string
		userID      string
	}

	cases := []struct {
		opts     []telemeter.Option
		expected result
	}{
		{opts: []telemeter.Option{
			telemeter.WithUserID("userID"),
		}, expected: result{
			userID:      "userID",
			anonymousID: "",
		}},
		{opts: []telemeter.Option{}, expected: result{
			userID:      "",
			anonymousID: "clientID",
		}},
		{opts: []telemeter.Option{
			telemeter.WithClient("anotherID", "clientType"),
		}, expected: result{
			userID:      "",
			anonymousID: "anotherID",
		}},
		{opts: []telemeter.Option{
			telemeter.WithUserID("userID"),
			telemeter.WithClient("anotherID", "clientType"),
		}, expected: result{
			userID:      "userID",
			anonymousID: "",
		}},
		{opts: []telemeter.Option{
			telemeter.WithClient("anotherID", "clientType"),
			telemeter.WithUserID("userID"),
		}, expected: result{
			userID:      "userID",
			anonymousID: "",
		}},
	}

	st := &segmentTelemeter{clientID: "clientID"}

	for _, c := range cases {
		opts := telemeter.ApplyOptions(c.opts)
		assert.Equal(t, c.expected.userID, st.getUserID(opts))
		assert.Equal(t, c.expected.anonymousID, st.getAnonymousID(opts))
	}
}
