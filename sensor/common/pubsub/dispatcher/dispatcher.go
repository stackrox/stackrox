package dispatcher

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

type Option func(*dispatcher)

func WithLaneConfigs(laneConfigs []pubsub.LaneConfig) Option {
	return func(ps *dispatcher) {
		if len(laneConfigs) == 0 {
			return
		}
		ps.laneConfigs = laneConfigs
	}
}

func NewDispatcher(opts ...Option) (*dispatcher, error) {
	ps := &dispatcher{
		lanes: make(map[pubsub.LaneID]pubsub.Lane),
	}
	for _, opt := range opts {
		opt(ps)
	}
	if err := ps.createLanes(); err != nil {
		ps.Stop()
		return nil, err
	}
	if len(ps.lanes) == 0 {
		return nil, errors.New("no lanes configured")
	}
	return ps, nil
}

type dispatcher struct {
	// TODO: Consider using a frozen structure to avoid mutexes
	laneLock    sync.RWMutex
	lanes       map[pubsub.LaneID]pubsub.Lane
	laneConfigs []pubsub.LaneConfig
}

func (d *dispatcher) Publish(event pubsub.Event) error {
	if event == nil {
		return errors.New("trying to publish a 'nil' event")
	}
	lane, err := d.getLane(event.Lane())
	if err != nil {
		return errors.Wrap(err, "unable to find lane for the given event")
	}
	return errors.Wrap(lane.Publish(event), "unable to publish event")
}

func (d *dispatcher) RegisterConsumer(consumerID pubsub.ConsumerID, topic pubsub.Topic, callback pubsub.EventCallback) error {
	if callback == nil {
		return errors.New("cannot register a 'nil' callback")
	}
	d.laneLock.RLock()
	defer d.laneLock.RUnlock()
	errList := errorhelpers.NewErrorList("register consumer")
	for _, lane := range d.lanes {
		if err := d.registerConsumerToLane(consumerID, topic, lane, callback); err != nil {
			errList.AddErrors(err)
		}
	}
	return errList.ToError()
}

func (d *dispatcher) RegisterConsumerToLane(consumerID pubsub.ConsumerID, topic pubsub.Topic, laneID pubsub.LaneID, callback pubsub.EventCallback) error {
	if callback == nil {
		return errors.New("cannot register a 'nil' callback")
	}
	lane, err := d.getLane(laneID)
	if err != nil {
		return errors.Errorf("lane with ID %q not found: %v", laneID, err)
	}
	return d.registerConsumerToLane(consumerID, topic, lane, callback)
}

func (d *dispatcher) Stop() {
	d.laneLock.RLock()
	defer d.laneLock.RUnlock()
	for _, lane := range d.lanes {
		lane.Stop()
	}
}

func (d *dispatcher) getLane(id pubsub.LaneID) (pubsub.Lane, error) {
	d.laneLock.RLock()
	defer d.laneLock.RUnlock()
	lane, ok := d.lanes[id]
	if !ok {
		return nil, errors.Errorf("unexpected lane %q", id.String())
	}
	return lane, nil
}

func (d *dispatcher) registerConsumerToLane(consumerID pubsub.ConsumerID, topic pubsub.Topic, lane pubsub.Lane, callback pubsub.EventCallback) error {
	return errors.Wrap(lane.RegisterConsumer(consumerID, topic, callback), "unable to register consumer")
}

func (d *dispatcher) createLanes() error {
	d.laneLock.Lock()
	defer d.laneLock.Unlock()
	errList := errorhelpers.NewErrorList("create lanes")
	for _, config := range d.laneConfigs {
		if _, ok := d.lanes[config.LaneID()]; ok {
			errList.AddError(errors.Errorf("duplicated lane %q configured", config.LaneID().String()))
			continue
		}
		lane := config.NewLane()
		if lane == nil {
			errList.AddErrors(errors.Errorf("unable to create lane %q", config.LaneID().String()))
			continue
		}
		d.lanes[config.LaneID()] = lane
	}
	return errList.ToError()
}
