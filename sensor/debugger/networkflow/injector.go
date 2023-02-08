package networkflow

import (
	"bytes"
	"os"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type FlowInjector struct {
	binFilePath string

	writeChan chan *sensor.NetworkConnectionInfoMessage
}

func NewFlowInjector(binPath string) *FlowInjector {
	return &FlowInjector{binFilePath: binPath}
}

func (f *FlowInjector) ReaderC() <-chan *sensor.NetworkConnectionInfoMessage {
	return f.writeChan
}

func (f *FlowInjector) RunInjector() {
	fileContent, err := os.ReadFile(f.binFilePath)
	if err != nil {
		log.Warnf("can't read injector binary file (%s) for Network Flows: %s", f.binFilePath, err)
		return
	}

	connectionMessages := bytes.Split(fileContent, []byte{0xFA, 0xFB, 0xFC, 0xFD})

	log.Infof("Fake Injector starting: %d messages found in binary file", len(connectionMessages))
	for idx, msg := range connectionMessages {
		m := &sensor.NetworkConnectionInfoMessage{}
		err := m.Unmarshal(msg)
		if err != nil {
			log.Warnf("failed to unmarshal message at position %d: %s", idx, err)
			continue
		}
		f.writeChan <- m
	}

	close(f.writeChan)
}
