package notifications

import (
	"fmt"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/types"
)

const alertChanSize = 100

var (
	log = logging.New("notifications")
)

// Processor takes in alerts and sends the notifications tied to that alert
type Processor struct {
	alertChan chan *v1.Alert
	notifiers map[string]types.Notifier
	database  db.NotifierStorage
	lock      sync.Mutex
}

// NewNotificationProcessor returns a new Processor
func NewNotificationProcessor(database db.NotifierStorage) (*Processor, error) {
	processor := &Processor{
		alertChan: make(chan *v1.Alert, alertChanSize),
		notifiers: make(map[string]types.Notifier),
		database:  database,
	}
	err := processor.initializeNotifiers()
	return processor, err
}

func (p *Processor) initializeNotifiers() error {
	protoNotifiers, err := p.database.GetNotifiers(&v1.GetNotifiersRequest{})
	if err != nil {
		return err
	}
	for _, protoNotifier := range protoNotifiers {
		notifierCreator, ok := notifiers.Registry[protoNotifier.Type]
		if !ok {
			return fmt.Errorf("Stored notifier type %v does not exist", protoNotifier.Type)
		}
		notifier, err := notifierCreator(protoNotifier)
		if err != nil {
			return fmt.Errorf("Error creating notifier with name %v and type %v: %v", protoNotifier.Name, protoNotifier.Type, err)
		}
		p.UpdateNotifier(notifier)
	}
	return nil
}

// Start begins the notification processor and is blocking
func (p *Processor) Start() {
	for {
		alert := <-p.alertChan

		p.lock.Lock()
		for _, name := range alert.Policy.Notifiers {
			notifier, exists := p.notifiers[name]
			if !exists {
				log.Errorf("Could not send notification to notifier %v for alert %v because it does not exist", name, alert.GetId())
				continue
			}
			if err := notifier.Notify(alert); err != nil {
				log.Errorf("Unable to send notification to %v for alert %v: %v", name, alert.GetId(), err)
			}
		}
		p.lock.Unlock()
	}
}

// Process pushs the alert into a channel to be processed
func (p *Processor) Process(alert *v1.Alert) {
	p.alertChan <- alert
}

// RemoveNotifier removes the in memory copy of the specified notifier
func (p *Processor) RemoveNotifier(name string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.notifiers, name)
}

// UpdateNotifier updates or adds the passed notifier into memory
func (p *Processor) UpdateNotifier(notifier types.Notifier) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.notifiers[notifier.ProtoNotifier().GetName()] = notifier
}
