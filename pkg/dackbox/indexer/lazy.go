package indexer

import (
	"fmt"
	"hash"
	"hash/fnv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	procInterval = 500 * time.Millisecond
	maxBatchSize = 500
)

var (
	log = logging.LoggerForModule()
)

// Acker is a function we call on keys that have been processed.
type Acker func(keys ...[]byte) error

// Lazy represents an interface for lazily indexing values that have been written to DackBox.
type Lazy interface {
	Start()
	Stop()
}

// NewLazy returns a new instance of a lazy indexer that reads in the values to index from the toIndex queue, indexes
// them with the given indexer, then acks indexed values with the given acker.
func NewLazy(toIndex queue.WaitableQueue, wrapper Wrapper, _ interface{}, acker Acker) Lazy {
	return &lazyImpl{
		wrapper:    wrapper,
		acker:      acker,
		toIndex:    toIndex,
		deduper:    make(map[string]uint64),
		hasher:     fnv.New64a(),
		buff:       NewBuffer(maxBatchSize),
		stopSignal: concurrency.NewSignal(),
	}
}

type lazyImpl struct {
	wrapper Wrapper
	acker   Acker
	toIndex queue.WaitableQueue
	deduper map[string]uint64
	hasher  hash.Hash64

	buff Buffer

	stopSignal concurrency.Signal
}

func (li *lazyImpl) Start() {
	go li.runIndexing()
}

func (li *lazyImpl) Stop() {
	li.stopSignal.Signal()
}

// No need for control logic since we always want this running with an instance of DackBox that uses lazy indexing.
func (li *lazyImpl) runIndexing() {
	ticker := time.NewTicker(procInterval)
	defer ticker.Stop()

	for {
		select {
		case <-li.stopSignal.Done():
			return

		// Don't wait more than the interval to index.
		case <-ticker.C:
			li.flush()

		// Collect items from the queue to index.
		case <-li.toIndex.NotEmpty().Done():
			li.consumeFromQueue()
		}
	}
}

func (li *lazyImpl) consumeFromQueue() {
	for li.toIndex.Length() > 0 {
		key, value, signal := li.toIndex.Pop()
		if key == nil && signal == nil {
			return
		}

		if key != nil {
			li.handleKeyValue(key, value)
		} else {
			li.buff.AddSignalToSend(signal)
		}

		// Don't ack more than the max at a time.
		if li.buff.IsFull() {
			li.flush()
		}
	}
}

func (li *lazyImpl) handleKeyValue(key []byte, value proto.Message) {
	indexedKey, indexedValue := li.wrapper.Wrap(key, value)
	if indexedKey == "" {
		log.Errorf("no wrapper registered for key: %q", key)
		return
	}
	li.buff.AddKeyToAck(key)
	li.buff.AddValueToIndex(indexedKey, indexedValue)
}

func (li *lazyImpl) flush() {
	// Index values in the buffer.
	if len(li.buff.ValuesToIndex()) > 0 {
		li.indexItems(li.buff.ValuesToIndex())
		li.ackKeys(li.buff.KeysToAck())
	}
	// Send signals in the buffer.
	if len(li.buff.SignalsToSend()) > 0 {
		li.sendSignals(li.buff.SignalsToSend())
	}
	li.buff.Reset()
}

func (li *lazyImpl) indexItems(_ map[string]interface{}) {}

func (li *lazyImpl) ackKeys(keysToAck [][]byte) {
	err := li.acker(keysToAck...)
	if err != nil {
		log.Errorf("unable to ack keys: %s, %v", printableKeys(keysToAck), err)
	}
}

func (li *lazyImpl) sendSignals(signals []*concurrency.Signal) {
	for _, signal := range signals {
		signal.Signal()
	}
}

// Helper for printing key values.
type printableKeys [][]byte

func (pk printableKeys) String() string {
	keys := make([]string, 0, len(pk))
	for _, key := range pk {
		keys = append(keys, fmt.Sprintf("%q", key))
	}
	return strings.Join(keys, ", ")
}
