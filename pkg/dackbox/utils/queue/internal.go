package queue

type internalQueue interface {
	push(interface{})
	pop() interface{}

	length() int
}

func newInternalQueue() internalQueue {
	return &internalQueueImpl{}
}

type internalQueueImpl struct {
	front       *keyNode
	back        *keyNode
	numElements int
}

func (q *internalQueueImpl) push(v interface{}) {
	newNode := &keyNode{
		val:  v,
		next: q.back,
	}
	// Set new node as back
	if q.back != nil {
		q.back.prev = newNode
	}
	q.back = newNode
	// If queue was empty, new node is now front as well.
	if q.front == nil {
		q.front = newNode
	}
	q.numElements++
}

func (q *internalQueueImpl) pop() interface{} {
	if q.front == nil {
		return nil
	}
	// Get key from front.
	ret := q.front.val
	// set front to it's previous value.
	q.front = q.front.prev
	// If front exists, reset it's next value to null.
	if q.front != nil {
		q.front.next = nil
	} else {
		q.back = nil
	}
	q.numElements--
	return ret
}

func (q *internalQueueImpl) length() int {
	return q.numElements
}

type keyNode struct {
	val  interface{}
	next *keyNode
	prev *keyNode
}
