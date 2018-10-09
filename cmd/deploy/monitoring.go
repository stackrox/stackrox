package main

import (
	"strings"

	"github.com/stackrox/rox/cmd/deploy/central"
	"github.com/stackrox/rox/pkg/logging"
)

var logger = logging.LoggerForModule()

type monitoringWrapper struct {
	Monitoring *central.MonitoringType
}

var monitoringMap = map[string]central.MonitoringType{
	"on-prem":         central.OnPrem,
	"none":            central.None,
	"stackrox-hosted": central.StackRoxHosted,
}

func (m *monitoringWrapper) String() string {
	return m.Monitoring.String()
}

func (m *monitoringWrapper) Set(input string) error {
	logger.Infof("Input: %s", input)
	val, ok := monitoringMap[strings.ToLower(input)]
	if !ok {
		*m.Monitoring = central.MonitoringType(central.OnPrem)
	} else {
		*m.Monitoring = val
	}
	return nil
}

func (m *monitoringWrapper) Type() string {
	return "monitoring"
}
