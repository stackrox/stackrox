package writer

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func TestValidateAdministrationEvent(t *testing.T) {
	cases := []struct {
		event *events.AdministrationEvent
		err   error
	}{
		{
			err: errox.InvalidArgs,
		},
		{
			event: &events.AdministrationEvent{},
			err:   errox.InvalidArgs,
		},
		{
			event: &events.AdministrationEvent{
				Domain:       "",
				Hint:         "only hint",
				Level:        0,
				Message:      "",
				ResourceID:   "",
				ResourceType: "",
				Type:         0,
			},
			err: errox.InvalidArgs,
		},
		{
			event: &events.AdministrationEvent{
				Domain:       "set",
				Level:        0,
				Message:      "set",
				ResourceID:   "set",
				ResourceType: "",
				Type:         0,
			},
			err: errox.InvalidArgs,
		},
		{
			event: &events.AdministrationEvent{
				Domain:       "set",
				Hint:         "set",
				Level:        0,
				Message:      "set",
				ResourceID:   "set",
				ResourceType: "set",
				Type:         0,
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("tc%d", i), func(t *testing.T) {
			err := validateAdministrationEvent(tc.event)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
