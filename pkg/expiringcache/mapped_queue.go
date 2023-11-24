package expiringcache

import (
	"container/list"
)

// A MappedQueue provides a mix of map and queue functionality, so that you can reference objects by key, but also
// fetch items in insertion order.
type mappedQueue interface {
	size() int

	push(key, value interface{})
	front() (key, value interface{})
	pop() (key, value interface{})

	remove(key interface{})
	removeAll()

	get(key interface{}) interface{}
	getAllValues() []interface{}
}

func newMappedQueue() mappedQueue {
	return &mappedQueueImpl{
		queue: list.New(),
		items: make(map[interface{}]*list.Element),
	}
}

type mappedQueueElement struct {
	key   interface{}
	value interface{}
}

type mappedQueueImpl struct {
	queue *list.List
	items map[interface{}]*list.Element
}

func (mq *mappedQueueImpl) size() int {
	return len(mq.items)
}

func (mq *mappedQueueImpl) push(key, value interface{}) {
	if mq.get(key) != nil {
		mq.remove(key)
	}
	// Add element to queue and map.
	listElement := mq.queue.PushBack(&mappedQueueElement{
		key:   key,
		value: value,
	})
	mq.items[key] = listElement
}

func (mq *mappedQueueImpl) front() (key, value interface{}) {
	frontElem := mq.queue.Front()
	if frontElem != nil {
		mqe := frontElem.Value.(*mappedQueueElement)
		key = mqe.key
		value = mqe.value
	}
	return
}

func (mq *mappedQueueImpl) pop() (key, value interface{}) {
	key, value = mq.front()
	if key != nil {
		mq.remove(key)
	}
	return
}

func (mq *mappedQueueImpl) get(key interface{}) interface{} {
	listElem, ok := mq.items[key]
	if !ok {
		return nil
	}
	element := listElem.Value.(*mappedQueueElement)
	return element.value
}

func (mq *mappedQueueImpl) getAllValues() []interface{} {
	if mq.queue.Len() == 0 {
		return nil
	}
	ret := make([]interface{}, 0, mq.queue.Len())
	for next := mq.queue.Front(); next != nil; next = next.Next() {
		ret = append(ret, next.Value.(*mappedQueueElement).value)
	}
	return ret
}

func (mq *mappedQueueImpl) remove(key interface{}) {
	listElem, ok := mq.items[key]
	if !ok {
		return
	}
	mq.queue.Remove(listElem)
	delete(mq.items, key)
}

func (mq *mappedQueueImpl) removeAll() {
	mq.queue = list.New()
	mq.items = make(map[interface{}]*list.Element)
}
