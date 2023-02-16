package phonehome

import (
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

type nilTelemeter struct{}

var _ telemeter.Telemeter = (*nilTelemeter)(nil)

func (t *nilTelemeter) Stop()                                                   {}
func (t *nilTelemeter) Identify(_ map[string]any, _ ...telemeter.Option)        {}
func (t *nilTelemeter) Track(_ string, _ map[string]any, _ ...telemeter.Option) {}
func (t *nilTelemeter) Group(_ map[string]any, _ ...telemeter.Option)           {}
