package phonehome

import (
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

type nilTelemeter struct{}

var _ telemeter.Telemeter = (*nilTelemeter)(nil)

func (t *nilTelemeter) Stop()                            {}
func (t *nilTelemeter) Identify(_ map[string]any)        {}
func (t *nilTelemeter) Track(_ string, _ map[string]any) {}
func (t *nilTelemeter) Group(_ string, _ map[string]any) {}

func (t *nilTelemeter) With(userID string) telemeter.Telemeter                    { return nil }
func (t *nilTelemeter) As(clientID string, clientType string) telemeter.Telemeter { return nil }
