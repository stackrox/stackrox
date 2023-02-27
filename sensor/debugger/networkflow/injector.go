package networkflow

import (
	"os"
	"path"

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
	entries, err := os.ReadDir(f.binFilePath)
	if err != nil {
		log.Warnf("can't read injector binary folder (%s) for Network Flows: %s", f.binFilePath, err)
		return
	}

	log.Infof("Found %d recorded network flows to send", len(entries))

	for _, entryFile := range entries {
		content, err := os.ReadFile(path.Join(f.binFilePath, entryFile.Name()))
		if err != nil {
			log.Warnf("failed to read file %s from network flow folder: %s", entryFile.Name(), err)
			continue
		}

		var message sensor.NetworkConnectionInfoMessage
		err = message.Unmarshal(content)
		if err != nil {
			log.Warnf("failed to unmarshal NetworkConnectionInfoMessage: %s", err)
			continue
		}

		f.writeChan <- &message
	}

	log.Infof("Finished processing all recorded network flows")

	close(f.writeChan)
}
