package graph

import "github.com/stackrox/stackrox/pkg/sync"

// History watches a graph's updates over time when they are applied through the tracker.
// You can 'Watch' the graphs state at points in time
type History interface {
	Hold() uint64
	View(at uint64) RGraph
	Release(at uint64)

	Apply(diff Modification)
}

// NewHistory returns a new instance of a history tracker for the input graph.
func NewHistory(master *Graph) History {
	return &historyTrackerImpl{
		updates: newQueue(),
		master:  master,
	}
}

type historyTrackerImpl struct {
	updates   *queue
	master    *Graph
	masterRef int32

	lock sync.RWMutex
}

// Hold drops an anchor at the current time-step so that the state of the graph at that time-step can be viewed.
// Returns the time-step.
func (v *historyTrackerImpl) Hold() uint64 {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.updates.back != nil {
		v.updates.back.refCount++
		return v.updates.back.timeStep
	}

	v.masterRef++
	return 0
}

// View returns a view of the graph at a given time-step.
// That time-step must be 'watched' in order to be viewed.
func (v *historyTrackerImpl) View(at uint64) RGraph {
	v.lock.RLock()
	defer v.lock.RUnlock()

	if at == 0 {
		return v.master
	}
	return NewCompositeGraph(v.master, v.updates.collect(at)...)
}

// Release removes the anchor added when Watch was called.
// This removed the ability to call View a the given time-step, and allows the history to be condensed into the master Graph.
func (v *historyTrackerImpl) Release(at uint64) {
	v.lock.Lock()
	defer v.lock.Unlock()

	if at == 0 && v.masterRef > 0 {
		v.masterRef--
	} else {
		v.updates.removeRef(at)
	}
	if v.masterRef == 0 {
		for _, update := range v.updates.prune() {
			update.Apply(v.master)
		}
	}
}

// Apply adds a change to the history of the graph at the current time-step.
// Takes ownership of the input 'diff', and returns the time-step it was added at.
func (v *historyTrackerImpl) Apply(diff Modification) {
	v.lock.Lock()
	defer v.lock.Unlock()

	v.updates.push(diff)
}

func newQueue() *queue {
	return &queue{
		currStep: 1,
	}
}

type queue struct {
	currStep uint64
	front    *queueNode
	back     *queueNode
}

func (q *queue) push(modification Modification) {
	q.pushAt(q.currStep, modification)
	q.currStep++
}

func (q *queue) removeRef(timeStep uint64) {
	for node := q.front; node != nil && node.timeStep <= timeStep; node = node.prev {
		if node.timeStep == timeStep {
			if node.refCount <= 0 {
				return
			}
			node.refCount--
			return
		}
	}
}

func (q *queue) collect(timeStep uint64) []Modification {
	var ret []Modification
	for node := q.front; node != nil && node.timeStep <= timeStep; node = node.prev {
		ret = append(ret, node.modification)
	}
	return ret
}

func (q *queue) prune() []Modification {
	var ret []Modification
	node := q.front
	for node != nil && node.refCount <= 0 {
		ret = append(ret, node.modification)
		q.pop()
		node = q.front
	}
	return ret
}

func (q *queue) pushAt(timeStep uint64, modification Modification) {
	newNode := &queueNode{
		timeStep:     timeStep,
		modification: modification,
		next:         q.back,
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
}

func (q *queue) pop() {
	if q.front == nil {
		return
	}
	// set front to it's previous value.
	q.front = q.front.prev
	// If front exists, reset it's next value to null.
	if q.front != nil {
		q.front.next = nil
	} else {
		q.back = nil
	}
}

type queueNode struct {
	timeStep     uint64
	refCount     int32
	modification Modification
	next         *queueNode
	prev         *queueNode
}
