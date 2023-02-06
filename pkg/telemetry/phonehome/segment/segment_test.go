package segment

import (
	"testing"

	segment "github.com/segmentio/analytics-go/v3"
	"github.com/stretchr/testify/assert"
)

func Test_getMessageType(t *testing.T) {
	track := segment.Track{
		Type: "Track",
	}

	assert.Equal(t, "Track", getMessageType(track))
}
