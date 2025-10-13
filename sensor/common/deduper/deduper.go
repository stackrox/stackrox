package deduper

import (
	"reflect"

	"github.com/pkg/errors"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/alert"
	"github.com/stackrox/rox/pkg/deduperkey"
	"github.com/stackrox/rox/pkg/logging"
	eventPkg "github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/pkg/sensor/hash"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/managedcentral"
	"github.com/stackrox/rox/sensor/common/messagestream"
	"github.com/stackrox/rox/sensor/common/metrics"
)

var (
	log = logging.LoggerForModule()
)

// deduper takes care of deduping sensor events.
type deduper struct {
	stream             messagestream.SensorMessageStream
	lastSent           map[deduperkey.Key]uint64
	centralState       set.StringSet
	unchangedIDs       set.StringSet
	synced             bool
	hasher             *hash.Hasher
	appendUnchangedIDs bool
}

// NewDedupingMessageStream wraps a SensorMessageStream and dedupes events. Other message types are forwarded as-is.
func NewDedupingMessageStream(stream messagestream.SensorMessageStream, deduperState map[deduperkey.Key]uint64, appendUnchangedIDs bool) messagestream.SensorMessageStream {
	if deduperState == nil {
		deduperState = make(map[deduperkey.Key]uint64)
	}

	centralOriginalState := set.NewStringSet()
	for k := range deduperState {
		centralOriginalState.Add(k.String())
	}

	return &deduper{
		stream:             stream,
		centralState:       centralOriginalState,
		lastSent:           deduperState,
		hasher:             hash.NewHasher(),
		unchangedIDs:       set.NewStringSet(),
		appendUnchangedIDs: appendUnchangedIDs,
	}
}

func (d *deduper) Send(msg *central.MsgFromSensor) error {
	eventMsg, ok := msg.GetMsg().(*central.MsgFromSensor_Event)
	if !ok || skipDeduping(eventMsg.Event) {
		if err := d.stream.Send(msg); err != nil {
			return errors.Wrap(err, "forwarding message without dedupe")
		}
		return nil
	}
	event := eventMsg.Event

	resourcesSynced := event.GetSynced()
	if d.appendUnchangedIDs && resourcesSynced != nil {
		d.synced = true
		log.Infof("Adding %d events as unchanged to sync event", len(d.unchangedIDs))
		resourcesSynced.UnchangedIds = d.unchangedIDs.AsSlice()
		metrics.IncrementTotalResourcesSyncSent(len(d.unchangedIDs))
		metrics.SetResourcesSyncedSize(msg.SizeVT())
	}

	// This filter works around race conditions in which image integrations may be initialized prior to CentralHello being received
	if managedcentral.IsCentralManaged() && event.GetImageIntegration() != nil {
		return nil
	}
	key := deduperkey.Key{
		ID:           event.GetId(),
		ResourceType: reflect.TypeOf(event.GetResource()),
	}
	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		priorLen := len(d.lastSent)
		delete(d.lastSent, key)
		// Do not send a remove message for something that has not been seen before
		// This also effectively dedupes REMOVE actions
		if priorLen == len(d.lastSent) {
			return nil
		}
		if err := d.stream.Send(msg); err != nil {
			return errors.Wrap(err, "sending remove resource message")
		}
		return nil
	}

	hashValue, ok := d.hasher.HashEvent(msg.GetEvent())
	if ok {
		// If the hash is valid, then check for deduping
		if d.lastSent[key] == hashValue {
			// If this is a SYNC event, we have to keep track of this event
			if d.appendUnchangedIDs && msg.GetEvent().GetAction() == central.ResourceAction_SYNC_RESOURCE {
				key := eventPkg.GetKeyFromMessage(msg)
				d.unchangedIDs.AddMatching(d.centralState.Contains, key)
			}
			return nil
		}
		event.SensorHashOneof = &central.SensorEvent_SensorHash{
			SensorHash: hashValue,
		}
		d.lastSent[key] = hashValue
	}

	if err := d.stream.Send(msg); err != nil {
		return errors.Wrap(err, "sending deduped message")
	}

	return nil
}

func skipDeduping(event *central.SensorEvent) bool {
	// We only dedupe event messages (excluding process indicators and runtime alerts which are always unique),
	// other messages get forwarded directly.
	// NodeInventory and node IndexReport shall never be deduped, as the vulnerability data in scanner might have been
	// updated and the resulting vulnerabilities may be different for the same message scanned at different time.
	return event.GetProcessIndicator() != nil ||
		alert.IsRuntimeAlertResult(event.GetAlertResults()) ||
		event.GetNodeInventory() != nil ||
		event.GetIndexReport() != nil ||
		event.GetVirtualMachine() != nil ||
		event.GetVirtualMachineIndexReport() != nil
}
