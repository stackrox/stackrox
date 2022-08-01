package wal

import (
	"container/list"

	"github.com/stackrox/rox/pkg/sync"
)

var (
	once  sync.Once
	acker MessageAcker
)

type Action int

const (
	Add Action = iota
	Remove
)

type MessageAcker interface {
	Insert(id string, hash uint64, action Action)
	Ack(value uint64) error
}

type listEntry struct {
	value  uint64
	id     string
	hash   uint64
	action Action
}

func Singleton() MessageAcker {
	once.Do(func() {
		acker = NewMessageAcker()
	})
	return acker
}

func NewMessageAcker() *messageAcker {
	return &messageAcker{
		queue: list.New(),
	}
}

type messageAcker struct {
	wal WAL

	lock  sync.Mutex
	value uint64
	queue *list.List
}

func (m *messageAcker) Insert(id string, hash uint64, action Action) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.value++
	m.queue.PushBack(&listEntry{
		value:  m.value,
		id:     id,
		hash:   hash,
		action: action,
	})
}

func (m *messageAcker) getAckedElements(value uint64) []*listEntry {
	m.lock.Lock()
	defer m.lock.Unlock()

	var ackedEntries []*listEntry
	for {
		front := m.queue.Front()
		if front == nil {
			break
		}
		le := front.Value.(*listEntry)
		if le.value > value {
			break
		}
		ackedEntries = append(ackedEntries, le)
		m.queue.Remove(front)
	}
	return ackedEntries
}

func (m *messageAcker) Ack(value uint64) error {
	entries := m.getAckedElements(value)

	// Flush outside of the lock because this is just an optimization
	for _, entry := range entries {
		switch entry.action {
		case Add:
			if err := m.wal.Insert(entry.id, entry.hash); err != nil {
				return err
			}
		case Remove:
			if err := m.wal.Delete(entry.id); err != nil {
				return err
			}
		}
	}
	return nil
}
