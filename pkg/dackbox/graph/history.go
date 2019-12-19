package graph

import (
	"sort"
)

// History watches a graph's updates over time when they are applied through the tracker.
// You can 'Watch' the graphs state at points in time
type History interface {
	Hold() uint64
	View(at uint64) RGraph
	Release(at uint64)

	StepForward() uint64
	Apply(diff Modification)
}

// NewHistory returns a new instance of a history tracker for the input graph.
func NewHistory(master *Graph) History {
	return &historyTrackerImpl{
		pushed: make(map[uint64]Modification),
		master: master,
	}
}

type historyTrackerImpl struct {
	watched  watchedBranchSet
	pushed   pushedBranchSet
	currStep uint64

	master *Graph
}

// Keep drops an anchor at the current time-step so that the state of the graph at that time-step can be viewed.
// Returns the time-step.
func (v *historyTrackerImpl) Hold() uint64 {
	v.watched.insert(v.currStep)
	return v.currStep
}

// View returns a view of the graph at a given time-step.
// That time-step must be 'watched' in order to be viewed.
func (v *historyTrackerImpl) View(at uint64) RGraph {
	if v.watched.find(at) < 0 {
		panic("cannot view an unwatched branch")
	}
	return NewCompositeGraph(v.master, v.pushed.peekPushedBefore(at)...)
}

// Discard removes the anchor added when Watch was called.
// This removed the ability to call View a the given time-step, and allows the history to be condensed into the master Graph.
func (v *historyTrackerImpl) Release(at uint64) {
	if !v.watched.remove(at) {
		return
	}
	for _, toApply := range v.pushed.popPushedBefore(v.watched.earliest()) {
		toApply.Apply(v.master)
	}
}

// Step moves forward one time-step in the history.
func (v *historyTrackerImpl) StepForward() uint64 {
	v.currStep++
	return v.currStep
}

// Apply adds a change to the history of the graph at the current time-step.
// Takes ownership of the input 'diff', and returns the time-step it was added at.
func (v *historyTrackerImpl) Apply(diff Modification) {
	at := v.currStep
	v.pushed.insert(at, diff)
}

// watchedBranchSet is the number of views of the history by the timestep the view corresponds to.
type watchedBranchSet struct {
	held []uint64
}

func (obs *watchedBranchSet) insert(at uint64) {
	obs.held = append(obs.held, at)
}

func (obs *watchedBranchSet) find(at uint64) int {
	if len(obs.held) == 0 {
		return -1
	}
	idx := sort.Search(len(obs.held), func(i int) bool {
		return obs.held[i] >= at
	})
	if idx >= 0 && idx < len(obs.held) && obs.held[idx] == at {
		return idx
	}
	return -1
}

func (obs *watchedBranchSet) remove(at uint64) bool {
	idx := obs.find(at)
	if idx >= 0 && idx < len(obs.held) && obs.held[idx] == at {
		obs.held = append(obs.held[:idx], obs.held[idx+1:]...)
		return true
	}
	return false
}

func (obs *watchedBranchSet) earliest() uint64 {
	if len(obs.held) == 0 {
		return 0
	}
	return obs.held[0]
}

// pushedBranchSet are the changes that have been pushed to the graph stored by time-step.
type pushedBranchSet map[uint64]Modification

func (cbs pushedBranchSet) insert(closedAt uint64, modification Modification) {
	cbs[closedAt] = modification
}

func (cbs pushedBranchSet) peekPushedBefore(at uint64) []Modification {
	closedTimes := cbs.timesPushedBefore(at)
	return cbs.getModifications(closedTimes)
}

func (cbs pushedBranchSet) popPushedBefore(at uint64) []Modification {
	closedTimes := cbs.timesPushedBefore(at)
	ret := cbs.getModifications(closedTimes)
	cbs.removeModifications(closedTimes)
	return ret
}

func (cbs pushedBranchSet) timesPushedBefore(at uint64) []uint64 {
	var closedTimes []uint64
	for closedAt := range cbs {
		if closedAt <= at {
			closedTimes = append(closedTimes, closedAt)
		}
	}
	sort.Slice(closedTimes, func(i, j int) bool {
		return closedTimes[i] < closedTimes[j]
	})
	return closedTimes
}

func (cbs pushedBranchSet) getModifications(atTimes []uint64) []Modification {
	var orderedModifications []Modification
	for _, next := range atTimes {
		orderedModifications = append(orderedModifications, cbs[next])
	}
	return orderedModifications
}

func (cbs pushedBranchSet) removeModifications(atTimes []uint64) {
	for _, next := range atTimes {
		delete(cbs, next)
	}
}
