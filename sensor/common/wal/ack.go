package wal

import (
	"container/list"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once  sync.Once
	acker MessageAcker
)

type MessageAcker interface {
	Insert(event *central.SensorEvent)
	Ack(value string) error
}

type listEntry struct {
	id     string
	hash   uint64
	action central.ResourceAction
}

type checkpointEntry struct {
	checkpoint string
}

func MessageAckerSingleton() MessageAcker {
	once.Do(func() {
		acker = NewMessageAcker()
	})
	return acker
}

func NewMessageAcker() *messageAcker {
	return &messageAcker{
		wal:   OpenWAL(),
		queue: list.New(),
	}
}

type messageAcker struct {
	wal WAL

	lock  sync.Mutex
	queue *list.List
}

func (m *messageAcker) Insert(event *central.SensorEvent) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if event.GetCheckpoint() != nil {
		m.queue.PushBack(&checkpointEntry{
			checkpoint: event.GetCheckpoint().GetId(),
		})
		return
	}
	log.Infof("Pushing back: %+v", event.GetResource())
	m.queue.PushBack(&listEntry{
		id:     event.GetId(),
		hash:   event.GetHash(),
		action: event.GetAction(),
	})
}

func (m *messageAcker) getAckedElements(value string) []*listEntry {
	m.lock.Lock()
	defer m.lock.Unlock()

	var ackedEntries []*listEntry
	for {
		front := m.queue.Front()
		if front == nil {
			break
		}
		m.queue.Remove(front)

		switch entryValue := front.Value.(type) {
		case *listEntry:
			ackedEntries = append(ackedEntries, entryValue)
		case *checkpointEntry:
			if entryValue.checkpoint == value {
				return ackedEntries
			}
			log.Errorf("found out of order checkpoints: %s vs %s. continuing", entryValue.checkpoint, value)
		default:
			panic("nope")
		}
	}
	return ackedEntries
}

func (m *messageAcker) Ack(value string) error {
	entries := m.getAckedElements(value)

	// Flush outside of the lock because this is just an optimization
	for _, entry := range entries {
		switch entry.action {
		case central.ResourceAction_REMOVE_RESOURCE:
			if err := m.wal.Delete(entry.id); err != nil {
				return err
			}
		default:
			if err := m.wal.Insert(entry.id, entry.hash); err != nil {
				return err
			}
		}
	}
	return nil
}
