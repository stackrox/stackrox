package events

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func TestAdministrationEvent_Validate(t *testing.T) {
	cases := []struct {
		event *AdministrationEvent
		err   error
	}{
		{
			err: errox.InvalidArgs,
		},
		{
			event: &AdministrationEvent{},
			err:   errox.InvalidArgs,
		},
		{
			event: &AdministrationEvent{
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
			event: &AdministrationEvent{
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
			event: &AdministrationEvent{
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
			err := tc.event.Validate()
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
