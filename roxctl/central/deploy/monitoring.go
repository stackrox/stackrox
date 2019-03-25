package deploy

import (
	"strings"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/roxctl/central/deploy/renderer"
)

var (
	log = logging.LoggerForModule()
)

type monitoringWrapper struct {
	Monitoring *renderer.MonitoringType
}

var monitoringMap = map[string]renderer.MonitoringType{
	"on-prem":         renderer.OnPrem,
	"none":            renderer.None,
	"stackrox-hosted": renderer.StackRoxHosted,
}

func (m *monitoringWrapper) String() string {
	return m.Monitoring.String()
}

func (m *monitoringWrapper) Set(input string) error {
	val, ok := monitoringMap[strings.ToLower(input)]
	if !ok {
		*m.Monitoring = renderer.MonitoringType(renderer.OnPrem)
	} else {
		*m.Monitoring = val
	}
	return nil
}

func (m *monitoringWrapper) Type() string {
	return "monitoring"
}
