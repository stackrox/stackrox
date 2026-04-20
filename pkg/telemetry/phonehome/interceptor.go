package phonehome

import (
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

// Interceptor is a function which will be called on every API call if none of
// the previous interceptors in the chain returned false.
// An Interceptor function may add custom properties to the props map so that
// they appear in the event.
type Interceptor func(rp *RequestParams, props map[string]any) bool

func (c *Client) track(rp *RequestParams) {
	if !c.IsActive() {
		return
	}
	c.interceptorsLock.RLock()
	defer c.interceptorsLock.RUnlock()
	if len(c.interceptors) == 0 {
		return
	}
	opts := append(c.WithGroups(),
		telemeter.WithUserID(c.HashUserAuthID(rp.UserID)))
	t := c.Telemeter()
	for event, funcs := range c.interceptors {
		props := map[string]any{}
		ok := true
		for _, interceptor := range funcs {
			if ok = interceptor(rp, props); !ok {
				break
			}
		}
		if ok {
			t.Track(event, props, opts...)
		}
	}
}
