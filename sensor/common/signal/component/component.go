package component

import (
	"io"
	"strings"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

const maxBufferSize = 10000

var (
	log = logging.LoggerForModule()
)

type componentImpl struct {
	common.SensorComponent

	processPipeline Pipeline
	indicators      <-chan *message.ExpiringMessage
	signalMessages  <-chan *storage.ProcessSignal
	processMessages <-chan *sensor.ProcessSignal
	writer          io.Writer

	stopper concurrency.Stopper
}

type Option func(*componentImpl)

// WithTraceWriter sets a trace writer that will write the messages received from collector.
func WithTraceWriter(writer io.Writer) Option {
	return func(cmp *componentImpl) {
		cmp.writer = writer
	}
}

func New(pipeline Pipeline, signalMessages <-chan *storage.ProcessSignal, processMessages <-chan *sensor.ProcessSignal, indicators <-chan *message.ExpiringMessage, opts ...Option) common.SensorComponent {
	cmp := &componentImpl{
		processPipeline: pipeline,
		indicators:      indicators,
		signalMessages:  signalMessages,
		processMessages: processMessages,
		writer:          nil,
		stopper:         concurrency.NewStopper(),
	}
	for _, o := range opts {
		o(cmp)
	}
	return cmp
}

func (c *componentImpl) Start() error {
	go c.run()
	return nil
}

func (c *componentImpl) Stop(_ error) {
	c.processPipeline.Shutdown()
	c.stopper.Client().Stop()
}

func (c *componentImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	c.processPipeline.Notify(e)
}

func (c *componentImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}
func (c *componentImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (c *componentImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return c.indicators
}

func (c *componentImpl) run() {
	defer c.stopper.Flow().ReportStopped()

	for {
		select {
		case msg := <-c.processMessages:
			c.processMsg(sensorIntoStorageSignal(msg))
		case msg := <-c.signalMessages:
			c.processMsg(msg)
		case <-c.stopper.Flow().StopRequested():
			log.Info("Shutting down signal component")
			return
		}
	}
}

func (c *componentImpl) processMsg(signal *storage.ProcessSignal) {
	signal.ExecFilePath = stringutils.OrDefault(signal.GetExecFilePath(), signal.GetName())
	if !isProcessSignalValid(signal) {
		log.Debugf("Invalid process signal: %+v", signal)
		return
	}
	if c.writer != nil {
		if data, err := signal.MarshalVT(); err == nil {
			if _, err := c.writer.Write(data); err != nil {
				log.Warnf("Error writing msg: %v", err)
			}
		} else {
			log.Warnf("Error marshalling  msg: %v", err)
		}
	}

	c.processPipeline.Process(signal)
}

// TODO(ROX-3281) this is a workaround for these collector issues
func isProcessSignalValid(signal *storage.ProcessSignal) bool {
	// Example: <NA> or sometimes a truncated variant
	if signal.GetExecFilePath() == "" || signal.GetExecFilePath()[0] == '<' {
		return false
	}
	if signal.GetName() == "" || signal.GetName()[0] == '<' {
		return false
	}
	if strings.HasPrefix(signal.GetExecFilePath(), "/proc/self") {
		return false
	}
	// Example: /var/run/docker/containerd/daemon/io.containerd.runtime.v1.linux/moby/8f79b77ac6785562e875cde2f087c49f1d4e4899f18a26d3739c47155668ec0b/run
	if strings.HasPrefix(signal.GetExecFilePath(), "/var/run/docker") {
		return false
	}
	return true
}

func sensorIntoStorageSignal(signal *sensor.ProcessSignal) *storage.ProcessSignal {
	if signal == nil {
		return nil
	}

	var lineage []*storage.ProcessSignal_LineageInfo
	if signal.LineageInfo != nil {
		lineage = make([]*storage.ProcessSignal_LineageInfo, 0, len(signal.LineageInfo))

		for _, l := range signal.LineageInfo {
			lineage = append(lineage, &storage.ProcessSignal_LineageInfo{
				ParentUid:          l.ParentUid,
				ParentExecFilePath: l.ParentExecFilePath,
			})
		}
	}

	return &storage.ProcessSignal{
		Id:           signal.Id,
		ContainerId:  signal.ContainerId,
		Time:         signal.CreationTime,
		Name:         signal.Name,
		Args:         signal.Args,
		ExecFilePath: signal.ExecFilePath,
		Pid:          signal.Pid,
		Uid:          signal.Uid,
		Gid:          signal.Gid,
		Scraped:      signal.Scraped,
		LineageInfo:  lineage,
	}
}
