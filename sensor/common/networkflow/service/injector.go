package service

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
)

var (
	injector CollectorFlowInjector
)

type CollectorFlowInjector interface {
	ReaderC() <-chan *sensor.NetworkConnectionInfoMessage
}

type noOpInjector struct{}

func (i *noOpInjector) ReaderC() <-chan *sensor.NetworkConnectionInfoMessage {
	c := make(chan *sensor.NetworkConnectionInfoMessage)
	defer close(c)
	return c
}

func init() {
	injector = &noOpInjector{}
}

func SetInjector(newInjector CollectorFlowInjector) {
	injector = newInjector
}

func GetInjector() CollectorFlowInjector {
	return injector
}
