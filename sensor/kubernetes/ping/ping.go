package ping

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

var (
	log = logging.LoggerForModule()

	_ common.SensorComponent = (*pingComponent)(nil)

	p    *pingComponent
	once sync.Once
)

// Singleton returns a singleton instance of the ping component.
func Singleton() common.SensorComponent {
	once.Do(func() {
		p = newPingComponent()
	})
	return p
}

func newPingComponent() *pingComponent {
	return &pingComponent{
		responsesC:   make(chan *message.ExpiringMessage),
		stopSignal:   concurrency.NewSignal(),
		pingInterval: env.PingInterval.DurationSetting(),
	}
}

type pingComponent struct {
	responsesC   chan *message.ExpiringMessage
	pingInterval time.Duration
	ticker       *time.Ticker
	stopSignal   concurrency.Signal
}

func (p *pingComponent) Notify(_ common.SensorComponentEvent) {}

func (p *pingComponent) Start() error {
	p.ticker = time.NewTicker(p.pingInterval)
	return nil
}

func (p *pingComponent) Stop(_ error) {
	p.ticker.Stop()
	p.stopSignal.Signal()
}

func (p *pingComponent) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.PingCap}
}

func (p *pingComponent) run() {
	for {
		select {
		case <-p.ticker.C:
			log.Debug("Sending ping message to Central.")
			p.responsesC <- message.New(&central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Ping{Ping: &central.SensorPing{}},
			})
		case <-p.stopSignal.Done():
			return
		}
	}
}

func (p *pingComponent) ProcessMessage(msg *central.MsgToSensor) error {
	if msg.GetPong() != nil {
		log.Debugf("Received Pong message from Central.")
	}
	return nil
}

func (p *pingComponent) ResponsesC() <-chan *message.ExpiringMessage {
	return p.responsesC
}
