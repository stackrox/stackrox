package telemeter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWith(t *testing.T) {
	opts := ApplyOptions([]Option{
		WithUserID("userID"),
		WithClient("clientID", "clientType"),
		WithGroupProperties("groupA", "groupA_id1", map[string]any{"key1": "value1"}),
		WithGroupProperties("groupA", "groupA_id2", map[string]any{"key-": "value-"}),
		WithGroupProperties("groupA", "groupA_id2", map[string]any{"key2": "value2"}),
		WithGroupProperties("groupB", "groupB_id", map[string]any{"key3": "value3"}),
	},
	)
	assert.Equal(t, "userID", opts.UserID)
	assert.Equal(t, "clientID", opts.ClientID)
	assert.Equal(t, "clientType", opts.ClientType)

	props := map[string]map[string]map[string]any{
		"groupA": {
			"groupA_id1": {"key1": "value1"},
			"groupA_id2": {"key2": "value2"},
		},
		"groupB": {
			"groupB_id": {"key3": "value3"},
		},
	}
	assert.Equal(t, props, opts.GroupProperties)
}
