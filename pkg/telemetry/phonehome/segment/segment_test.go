package segment

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

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

func Test_makeContext(t *testing.T) {
	opts := telemeter.ApplyOptions([]telemeter.Option{
		telemeter.WithUserID("userID"),
		telemeter.WithClient("clientID", "clientType"),
		telemeter.WithGroups("groupA", "groupA_id1"),
		telemeter.WithGroups("groupA", "groupA_id2"),
		telemeter.WithGroups("groupB", "groupB_id"),
	})

	s := segmentTelemeter{clientType: "test"}

	ctx := s.makeContext(opts)
	assert.Equal(t, "clientID", ctx.Device.Id)
	assert.Equal(t, "clientType", ctx.Device.Type)

	require.Contains(t, ctx.Extra, "groups")
	require.Contains(t, ctx.Extra["groups"], "groupA")
	assert.Contains(t, ctx.Extra["groups"], "groupB")
	groups := ctx.Extra["groups"].(map[string][]string)
	assert.ElementsMatch(t, []string{"groupA_id1", "groupA_id2"}, groups["groupA"])

	ctx = s.makeContext(telemeter.ApplyOptions([]telemeter.Option{}))
	assert.Equal(t, "test Server", ctx.Device.Type)

	ctx = s.makeContext(telemeter.ApplyOptions([]telemeter.Option{
		telemeter.WithClient("clientID", "clientType")}))
	assert.Equal(t, "clientType Server", ctx.Device.Type)
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

func Test_Group(t *testing.T) {
	var i int32

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&i, 1)
	}))

	tt := NewTelemeter("test-key", s.URL, "client-id", "client-type", 0, 1)

	tt.Group(nil, telemeter.WithGroups("Test", "test-group-id"))
	tt.Stop()
	s.Close()
	assert.Equal(t, int32(1), i, "Group call had to issue 1 message")
}

func Test_GroupWithProps(t *testing.T) {
	var i int32

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&i, 1)
	}))

	tt := NewTelemeter("test-key", s.URL, "client-id", "client-type", 0, 1)

	ch := make(chan time.Time, 2)
	ch <- time.Time{}
	ch <- time.Time{}

	ti := &time.Ticker{C: ch}
	options := telemeter.ApplyOptions(
		[]telemeter.Option{telemeter.WithGroups("Test", "test-group-id")},
	)
	tt.group(map[string]any{"key": "value"}, options)
	tt.groupFix(options, ti)
	tt.Stop()
	s.Close()
	assert.Equal(t, int32(4), i, "Group call had to issue 4 messages")
}

func Test_makeMessageID(t *testing.T) {
	tt := NewTelemeter("test-key", "url", "client-id", "client-type", 0, 1)

	props := map[string]any{
		"key":  "value",
		"int":  42,
		"bool": true,
	}
	salty := func(options ...telemeter.Option) *telemeter.CallOptions {
		opts := &telemeter.CallOptions{MessageIDPrefix: "test"}
		for _, o := range options {
			o(opts)
		}
		return opts
	}

	t.Run("Same ID with same input", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, salty())
		id2 := tt.makeMessageID("test event", props, salty())
		assert.Len(t, id1, 21)
		assert.Equal(t, id1, id2)
	})
	t.Run("Different ID with different props", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, salty())
		props["bool"] = false
		id2 := tt.makeMessageID("test event", props, salty())
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Different ID with different user props", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, salty(telemeter.WithTraits(map[string]any{"key": "same"})))
		id2 := tt.makeMessageID("test event", props, salty(telemeter.WithTraits(map[string]any{"key": "different"})))
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Different ID with different salt", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, salty())
		id2 := tt.makeMessageID("test event", props, salty(telemeter.WithMessageIDPrefix("different")))
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Different ID with different event", func(t *testing.T) {
		id1 := tt.makeMessageID("test event 1", props, salty())
		id2 := tt.makeMessageID("test event 2", props, salty())
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Different ID with different user", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, salty(telemeter.WithUserID("same")))
		id2 := tt.makeMessageID("test event", props, salty(telemeter.WithUserID("different")))
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Different ID with different client and user", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, salty(telemeter.WithClient("same", "same")))
		id2 := tt.makeMessageID("test event", props, salty(telemeter.WithUserID("same")))
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Empty ID with no salt added", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, &telemeter.CallOptions{})
		assert.Empty(t, id1)
		id1 = tt.makeMessageID("test event", nil, nil)
		assert.Empty(t, id1)
	})
}
