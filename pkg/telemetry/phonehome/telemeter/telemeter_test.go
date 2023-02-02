package telemeter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWith(t *testing.T) {
	opts := ApplyOptions([]Option{
		WithUserID("userID"),
		WithClient("clientID", "clientType"),
		WithGroup("groupA", "groupA_id1"),
		WithGroup("groupA", "groupA_id2"),
		WithGroupProperties("groupA_id", map[string]any{"key1": "value1"}),
		WithGroupProperties("groupB_id", map[string]any{"key2": "value2"}),
	},
	)
	assert.Equal(t, "userID", opts.UserID)
	assert.Equal(t, "clientID", opts.ClientID)
	assert.Equal(t, "clientType", opts.ClientType)
	assert.Len(t, opts.Groups, 1)
	assert.Len(t, opts.Groups["groupA"], 2)
	assert.Equal(t, "groupA_id1", opts.Groups["groupA"][0])
	assert.Equal(t, "groupA_id2", opts.Groups["groupA"][1])
	assert.Len(t, opts.GroupProperties["groupA_id"], 1)
	assert.Len(t, opts.GroupProperties["groupB_id"], 1)
	assert.Equal(t, "value1", opts.GroupProperties["groupA_id"]["key1"])
	assert.Equal(t, "value2", opts.GroupProperties["groupB_id"]["key2"])
}
