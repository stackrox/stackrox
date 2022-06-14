package indexer

import (
	"github.com/stackrox/stackrox/pkg/concurrency"
)

// Buffer is a buffer used by the lazy indexer to collect values which require updating in the index.
type Buffer interface {
	AddKeyToAck(key []byte)
	AddValueToIndex(indexedKey string, indexedValue interface{})
	AddSignalToSend(signal *concurrency.Signal)
	Reset()

	IsFull() bool

	ValuesToIndex() map[string]interface{}
	KeysToAck() [][]byte
	SignalsToSend() []*concurrency.Signal
}

// bufferImpl is an implementation of Buffer.
type bufferImpl struct {
	valuesToIndex map[string]interface{}
	keysToAck     [][]byte
	signalsToSend []*concurrency.Signal

	maxSize int
}

// NewBuffer returns an implementation of Buffer that is full when the number of keys = maxSize.
func NewBuffer(maxSize int) Buffer {
	return &bufferImpl{
		valuesToIndex: make(map[string]interface{}, maxSize),
		keysToAck:     make([][]byte, 0, maxSize),
		maxSize:       maxSize,
	}
}

func (ib *bufferImpl) ValuesToIndex() map[string]interface{} {
	return ib.valuesToIndex
}

func (ib *bufferImpl) KeysToAck() [][]byte {
	return ib.keysToAck
}

func (ib *bufferImpl) SignalsToSend() []*concurrency.Signal {
	return ib.signalsToSend
}

func (ib *bufferImpl) IsFull() bool {
	return len(ib.keysToAck) >= ib.maxSize || len(ib.signalsToSend) > ib.maxSize
}

func (ib *bufferImpl) Reset() {
	ib.keysToAck = ib.keysToAck[:0]
	ib.valuesToIndex = make(map[string]interface{})
	ib.signalsToSend = nil
}

func (ib *bufferImpl) AddKeyToAck(key []byte) {
	ib.keysToAck = append(ib.keysToAck, key)
}

func (ib *bufferImpl) AddValueToIndex(indexedKey string, indexedValue interface{}) {
	ib.valuesToIndex[indexedKey] = indexedValue
}

func (ib *bufferImpl) AddSignalToSend(signal *concurrency.Signal) {
	ib.signalsToSend = append(ib.signalsToSend, signal)
}
