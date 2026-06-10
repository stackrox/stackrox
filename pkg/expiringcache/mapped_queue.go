package expiringcache

import (
	"container/list"
)

// A MappedQueue provides a mix of map and queue functionality, so that you can reference objects by key, but also
// fetch items in insertion order.
type mappedQueue interface {
	size() int

	push(key, value any)
	front() (key, value any)
	pop() (key, value any)

	remove(key any)
	removeAll()

	get(key any) any
	getAllValues() []any
}

func newMappedQueue() mappedQueue {
	return &mappedQueueImpl{
		queue: list.New(),
		items: make(map[any]*list.Element),
	}
}

type mappedQueueElement struct {
	key   any
	value any
}

type mappedQueueImpl struct {
	queue *list.List
	items map[any]*list.Element
}

func (mq *mappedQueueImpl) size() int {
	return len(mq.items)
}

func (mq *mappedQueueImpl) push(key, value any) {
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

func (mq *mappedQueueImpl) front() (key, value any) {
	frontElem := mq.queue.Front()
	if frontElem != nil {
		mqe := frontElem.Value.(*mappedQueueElement)
		key = mqe.key
		value = mqe.value
	}
	return
}

func (mq *mappedQueueImpl) pop() (key, value any) {
	key, value = mq.front()
	if key != nil {
		mq.remove(key)
	}
	return
}

func (mq *mappedQueueImpl) get(key any) any {
	listElem, ok := mq.items[key]
	if !ok {
		return nil
	}
	element := listElem.Value.(*mappedQueueElement)
	return element.value
}

func (mq *mappedQueueImpl) getAllValues() []any {
	if mq.queue.Len() == 0 {
		return nil
	}
	ret := make([]any, 0, mq.queue.Len())
	for next := mq.queue.Front(); next != nil; next = next.Next() {
		ret = append(ret, next.Value.(*mappedQueueElement).value)
	}
	return ret
}

func (mq *mappedQueueImpl) remove(key any) {
	listElem, ok := mq.items[key]
	if !ok {
		return
	}
	mq.queue.Remove(listElem)
	delete(mq.items, key)
}

func (mq *mappedQueueImpl) removeAll() {
	mq.queue = list.New()
	mq.items = make(map[any]*list.Element)
}
