package tester

import (
	"sync"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	log                         = logging.LoggerForModule()
	once                        sync.Once
	eventPipelineTesterInstance *eventPipelineTester
)

type Tester interface {
	Send(*component.MsgToTester)
	SendAlerts(string, *central.MsgFromSensor)
}

func GetEventPipelineTester() *eventPipelineTester {
	if !env.ResyncTester.BooleanSetting() {
		return nil
	}
	once.Do(func() {
		eventPipelineTesterInstance = newEventPipelineTester()
	})
	return eventPipelineTesterInstance
}

func newEventPipelineTester() *eventPipelineTester {
	return &eventPipelineTester{
		stopper:     concurrency.NewStopper(),
		msgC:        make(chan *component.MsgToTester),
		eventsStore: newEventStore(),
		alertStore:  newAlertStore(),
	}
}

type eventPipelineTester struct {
	stopper     concurrency.Stopper
	msgC        chan *component.MsgToTester
	eventsStore *eventStore
	alertStore  *alertStore
	started     bool
}

func (t *eventPipelineTester) Send(msg *component.MsgToTester) {
	if t == nil {
		log.Debug("tester not initialize")
		return
	}
	if t.started {
		t.msgC <- msg
	}
}

func (t *eventPipelineTester) SendAlerts(id string, results *central.MsgFromSensor) {
	if t == nil {
		log.Debug("tester not initialize")
		return
	}
	if t.started {
		go t.updateAndCompareAlerts(id, results)
	}
}

func (t *eventPipelineTester) updateAndCompareAlerts(id string, results *central.MsgFromSensor) {
	if alerts := t.alertStore.get(id); alerts != nil {
		if alerts.IsResyncEvent() {
			log.Debugf("Deployment: %s Number of alerts: %d == %d", alerts.GetMsgToCentral().GetEvent().GetAlertResults().GetDeploymentId(),
				len(alerts.GetMsgToCentral().GetEvent().GetAlertResults().GetAlerts()),
				len(results.GetEvent().GetAlertResults().GetAlerts()))
		} else {
			alerts.MsgToCentral = results
			t.alertStore.upsert(alerts)
		}
	}
}

func (t *eventPipelineTester) Start() error {
	t.started = true
	go t.run()
	return nil
}

type alertStore struct {
	lock   sync.RWMutex
	alerts map[string]*component.MsgToTester
}

func newAlertStore() *alertStore {
	return &alertStore{
		alerts: make(map[string]*component.MsgToTester),
	}
}

func (a *alertStore) upsert(event *component.MsgToTester) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.alerts[event.GetId()] = event
}

func (a *alertStore) delete(event *component.MsgToTester) {
	a.lock.Lock()
	defer a.lock.Unlock()
	delete(a.alerts, event.GetId())
}

func (a *alertStore) get(id string) *component.MsgToTester {
	a.lock.Lock()
	defer a.lock.Unlock()
	return a.alerts[id]
}

type eventStore struct {
	lock   sync.RWMutex
	events map[string]*component.MsgToTester
}

func newEventStore() *eventStore {
	return &eventStore{
		events: make(map[string]*component.MsgToTester),
	}
}

func (es *eventStore) upsert(event *component.MsgToTester) {
	es.lock.Lock()
	defer es.lock.Unlock()
	es.events[event.GetId()] = event
}

func (es *eventStore) delete(event *component.MsgToTester) {
	es.lock.Lock()
	defer es.lock.Unlock()
	delete(es.events, event.GetId())
}

func (es *eventStore) get(id string) *component.MsgToTester {
	es.lock.Lock()
	defer es.lock.Unlock()
	return es.events[id]
}

func (t *eventPipelineTester) run() {
	defer t.stopper.Flow().ReportStopped()
	for {
		select {
		case msg, ok := <-t.msgC:
			if !ok {
				return
			}
			// log.Debugf("Msg to be sent: Id: %s, Resync: %s", msg.GetId(), msg.GetResourceVersion())
			// log.Debugf("MsgToCentral: %+v", msg.MsgToCentral)
			t.processEvent(msg)
		case <-t.stopper.Flow().StopRequested():
			return
		}
	}
}

func (t *eventPipelineTester) processSensorEvent(event *component.MsgToTester) {
	if oldEvent := t.eventsStore.get(event.GetId()); oldEvent != nil {
		if event.GetResourceVersion() == oldEvent.GetResourceVersion() {
			log.Debugf("Resync event")
			if event.GetMsgToCentral().GetEvent().GetDeployment() != nil {
				deployment := event.GetMsgToCentral().GetEvent().GetDeployment()
				oldDeployment := oldEvent.GetMsgToCentral().GetEvent().GetDeployment()
				log.Debug("Equal: ", deployment.GetHash() == oldDeployment.GetHash())
			}
		}
	} else {
		t.eventsStore.upsert(event)
	}
}

func (t *eventPipelineTester) processSensorAlerts(event *component.MsgToTester) {
	if oldEvent := t.alertStore.get(event.GetId()); oldEvent != nil {
		if event.GetResourceVersion() == oldEvent.GetResourceVersion() {
			log.Debug("Alert Resync event")
			oldEvent.ResyncEvent = true
			t.alertStore.upsert(oldEvent)
		}
	} else {
		t.alertStore.upsert(event)
	}
}

func (t *eventPipelineTester) processEvent(event *component.MsgToTester) {
	if event.GetMsgToCentral() != nil {
		if event.GetMsgToCentral().GetEvent() != nil {
			if event.GetMsgToCentral().GetEvent().GetAlertResults() != nil {
				t.processSensorAlerts(event)
			} else {
				t.processSensorEvent(event)
			}
		}
	}
}

func (t *eventPipelineTester) Stop(_ error) {
	t.stopper.Client().Stop()
}

func (t *eventPipelineTester) Notify(common.SensorComponentEvent) {}

func (t *eventPipelineTester) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (t *eventPipelineTester) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (t *eventPipelineTester) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}
