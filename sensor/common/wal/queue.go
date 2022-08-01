package wal

import (
	"hash"
	"hash/fnv"

	"github.com/mitchellh/hashstructure"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/messagestream"
)

var (
	log = logging.LoggerForModule()
)

// deduper takes care of deduping sensor events.
type deduper struct {
	stream messagestream.SensorMessageStream
	wal    WAL

	lastSent map[string]uint64
	hasher   hash.Hash64
}

// NewDataStream wraps a SensorMessageStream and dedupes events. Other message types are forwarded as-is.
func NewDataStream(stream messagestream.SensorMessageStream) messagestream.SensorMessageStream {
	return &deduper{
		stream:   stream,
		lastSent: make(map[string]uint64),
		hasher:   fnv.New64a(),
	}
}

func (d *deduper) Send(msg *central.MsgFromSensor) error {
	if err := d.wal.Insert(msg.GetEvent().GetId(), msg.GetEvent().GetHash()); err != nil {
		return err
	}
	return d.stream.Send(msg)

	eventMsg, ok := msg.Msg.(*central.MsgFromSensor_Event)
	if !ok || eventMsg.Event.GetProcessIndicator() != nil || isRuntimeAlert(msg) {
		// We only dedupe event messages (excluding process indicators and runtime alerts which are always unique), other messages get forwarded directly.
		return d.stream.Send(msg)
	}
	event := eventMsg.Event
	key := event.GetId()
	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		priorLen := len(d.lastSent)
		delete(d.lastSent, key)
		// Do not send a remove message for something that has not been seen before
		// This also effectively dedupes REMOVE actions
		if priorLen == len(d.lastSent) {
			return nil
		}
		return d.stream.Send(msg)
	}

	d.hasher.Reset()
	hashValue, err := hashstructure.Hash(event.GetResource(), &hashstructure.HashOptions{
		TagName: "sensorhash",
		Hasher:  d.hasher,
	})
	utils.Should(err)

	if d.lastSent[key] == hashValue {
		return nil
	}

	if err := d.stream.Send(msg); err != nil {
		return err
	}
	d.lastSent[key] = hashValue

	return nil
}
