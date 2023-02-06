package telemeter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWith(t *testing.T) {
	opts := ApplyOptions([]Option{
		WithUserID("userID"),
		WithClient("clientID", "clientType"),
	},
	)
	assert.Equal(t, "userID", opts.UserID)
	assert.Equal(t, "clientID", opts.ClientID)
	assert.Equal(t, "clientType", opts.ClientType)
}
