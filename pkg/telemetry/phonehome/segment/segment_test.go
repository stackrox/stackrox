package segment

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	segment "github.com/segmentio/analytics-go/v3"
	"github.com/stackrox/rox/pkg/expiringcache"
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
		telemeter.WithClient("clientID", "clientType", "clientVersion"),
		telemeter.WithGroups("groupA", "groupA_id1"),
		telemeter.WithGroups("groupA", "groupA_id2"),
		telemeter.WithGroups("groupB", "groupB_id"),
	})

	s := segmentTelemeter{clientType: "test"}

	ctx := s.makeContext(opts)
	assert.Equal(t, "clientID", ctx.Device.Id)
	assert.Equal(t, "clientType", ctx.Device.Type)
	assert.Equal(t, "clientVersion", ctx.Device.Version)

	require.Contains(t, ctx.Extra, "groups")
	require.Contains(t, ctx.Extra["groups"], "groupA")
	assert.Contains(t, ctx.Extra["groups"], "groupB")
	groups := ctx.Extra["groups"].(map[string][]string)
	assert.ElementsMatch(t, []string{"groupA_id1", "groupA_id2"}, groups["groupA"])

	ctx = s.makeContext(telemeter.ApplyOptions([]telemeter.Option{}))
	assert.Equal(t, "test Server", ctx.Device.Type)

	ctx = s.makeContext(telemeter.ApplyOptions([]telemeter.Option{
		telemeter.WithClient("clientID", "clientType", "clientVersion")}))
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
			telemeter.WithClient("anotherID", "clientType", "clientVersion"),
		}, expected: result{
			userID:      "",
			anonymousID: "anotherID",
		}},
		{opts: []telemeter.Option{
			telemeter.WithUserID("userID"),
			telemeter.WithClient("anotherID", "clientType", "clientVersion"),
		}, expected: result{
			userID:      "userID",
			anonymousID: "",
		}},
		{opts: []telemeter.Option{
			telemeter.WithClient("anotherID", "clientType", "clientVersion"),
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

	tt := NewTelemeter("test-key", s.URL, "client-id", "client-type", "client-version", 0, 1)

	tt.Group(telemeter.WithGroups("Test", "test-group-id"))
	tt.Stop()
	s.Close()
	assert.Equal(t, int32(1), i, "Group call had to issue 1 message")
}

func Test_GroupWithProps(t *testing.T) {
	var i int32

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&i, 1)
	}))

	tt := NewTelemeter("test-key", s.URL, "client-id", "client-type", "client-version", 0, 1)

	ch := make(chan time.Time, 2)
	ch <- time.Time{}
	ch <- time.Time{}

	ti := &time.Ticker{C: ch}
	options := telemeter.ApplyOptions(
		[]telemeter.Option{telemeter.WithGroups("Test", "test-group-id")},
	)
	tt.group("id", options)
	tt.groupFix(options, ti)
	tt.Stop()
	s.Close()
	assert.Equal(t, int32(4), i, "Group call had to issue 4 messages")
}

func Test_makeMessageID(t *testing.T) {
	tt := NewTelemeter("test-key", "url", "client-id", "client-type", "client-version", 0, 1)

	props := map[string]any{
		"key":  "value",
		"int":  42,
		"bool": true,
	}
	prefixed := func(options ...telemeter.Option) *telemeter.CallOptions {
		opts := &telemeter.CallOptions{MessageIDPrefix: "test"}
		for _, o := range options {
			o(opts)
		}
		return opts
	}

	t.Run("Same ID with same input", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, prefixed())
		id2 := tt.makeMessageID("test event", props, prefixed())
		assert.Equal(t, "test-490495b839acbeda", id1)
		assert.Equal(t, id1, id2)
	})
	t.Run("Different ID with different props", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, prefixed())
		props["bool"] = false
		id2 := tt.makeMessageID("test event", props, prefixed())
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Different ID with different user props", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, prefixed(telemeter.WithTraits(map[string]any{"key": "same"})))
		id2 := tt.makeMessageID("test event", props, prefixed(telemeter.WithTraits(map[string]any{"key": "different"})))
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Different ID with different prefix", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, prefixed())
		id2 := tt.makeMessageID("test event", props, prefixed(telemeter.WithNoDuplicates("different")))
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Different ID with different event", func(t *testing.T) {
		id1 := tt.makeMessageID("test event 1", props, prefixed())
		id2 := tt.makeMessageID("test event 2", props, prefixed())
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Different ID with different user", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, prefixed(telemeter.WithUserID("same")))
		id2 := tt.makeMessageID("test event", props, prefixed(telemeter.WithUserID("different")))
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Different ID with different client and user", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, prefixed(telemeter.WithClient("same", "same", "same")))
		id2 := tt.makeMessageID("test event", props, prefixed(telemeter.WithUserID("same")))
		assert.NotEqual(t, id1, id2)
	})
	t.Run("Empty ID with no prefix added", func(t *testing.T) {
		id1 := tt.makeMessageID("test event", props, &telemeter.CallOptions{})
		assert.Empty(t, id1)
		id1 = tt.makeMessageID("test event", nil, nil)
		assert.Empty(t, id1)
	})
}

type testClock struct {
	t time.Time
}

func (tc *testClock) Now() time.Time {
	return tc.t
}

func (tc *testClock) add(d time.Duration) {
	tc.t = tc.t.Add(d)
}

func initTestCache() *testClock {
	var tc testClock = testClock{time.Now()}
	// Override the global cache with custom clock for testing purposes:
	expiringIDCache = expiringcache.NewExpiringCacheWithClock(&tc, 1*time.Hour, expiringcache.UpdateExpirationOnGets[string, bool])
	return &tc
}

func Test_isDuplicate(t *testing.T) {
	tc := initTestCache()

	assert.False(t, isDuplicate("id1"))
	assert.False(t, isDuplicate("id2"))
	assert.True(t, isDuplicate("id1"))
	assert.True(t, isDuplicate("id2"))

	tc.add(2 * time.Hour)
	assert.False(t, isDuplicate("id1"), "Should forget id1 after 1 hour")
	assert.False(t, isDuplicate("id2"), "Should forget id2 after 1 hour")

	tc.add(30 * time.Minute)
	assert.True(t, isDuplicate("id1"), "Should remember id1 after 30 minutes")
	tc.add(40 * time.Minute)
	assert.True(t, isDuplicate("id1"), "Previous call should refresh the record")
}

func TestTrackWithNoDuplicates(t *testing.T) {
	tc := initTestCache()

	var i int32

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&i, 1)
	}))
	defer s.Close()

	t.Run("only one message", func(t *testing.T) {
		tt := NewTelemeter("test-key", s.URL, "client-id", "client-type", "client-version", 0, 1)
		for i := 0; i < 5; i++ {
			tt.Track("test event", nil, telemeter.WithNoDuplicates("today"))
		}
		tt.Stop()
		assert.Equal(t, int32(1), i, "Track calls had to issue 1 message")
	})
	t.Run("one message after cache expiry", func(t *testing.T) {
		tc.add(time.Hour)
		tc.add(time.Second)
		tt := NewTelemeter("test-key", s.URL, "client-id", "client-type", "client-version", 0, 1)
		for i := 0; i < 5; i++ {
			tt.Track("test event", nil, telemeter.WithNoDuplicates("today"))
		}
		tt.Stop()
		assert.Equal(t, int32(2), i, "Track calls had to issue one more message")
	})
	t.Run("different prefix", func(t *testing.T) {
		tt := NewTelemeter("test-key", s.URL, "client-id", "client-type", "client-version", 0, 1)
		for i := 0; i < 5; i++ {
			tt.Track("test event", nil, telemeter.WithNoDuplicates("tomorrow"))
		}
		tt.Stop()
		assert.Equal(t, int32(3), i, "Track calls had to issue one more message")
	})
	t.Run("different event", func(t *testing.T) {
		tt := NewTelemeter("test-key", s.URL, "client-id", "client-type", "client-version", 0, 1)
		for i := 0; i < 5; i++ {
			tt.Identify(telemeter.WithNoDuplicates("tomorrow"))
		}
		tt.Stop()
		assert.Equal(t, int32(4), i, "Identify calls had to issue one more message")
	})
}
