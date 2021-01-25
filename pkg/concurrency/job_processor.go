package concurrency

import (
	"container/list"
	"runtime"

	"github.com/pkg/errors"
)

var (
	// ErrJobProcessorStopped is the error returned by AddJob when the job processor has been stopped.
	ErrJobProcessorStopped = errors.New("job processor has been stopped")
)

// A JobProcessor allows callers to submit pieces of work that will be executed concurrently by a set of workers.
// When specifying a job, in addition to the function to be executed itself, callers can provide a function that
// specifies conflicts between the incoming job and whatever other jobs exist in the processor.
// The processor guarantees that when a job executes, no job it conflicts with is executed at the same time.
// Further, it guarantees that if one call to AddJob returns before another call to AddJob, and the two jobs
// conflict, the first job will execute first.
// This is useful when, for example, trying to ensure that jobs that touch the same data are not executed at the same
// time.
type JobProcessor interface {
	// AddJob adds a job to the processor. The execute func represents what actually gets executed.
	// Each job also comes with some arbitrary caller-supplied metadata.
	// The caller can also provide a conflictsWith function.
	// If not nil, this function is called with the caller-supplied metadata of each job
	// that is currently pending in the processor, and whichever job it returns true for
	// is marked as conflicting with this job.
	// It is generally expected that the conflictsWith function is symmetric
	// although this is not enforced.
	// Jobs are added in the order in which they were received, and if two jobs conflict,
	// the job received first will execute before the second.
	// It is safe to call AddJob simultaneously from multiple goroutines, although it is then unspecified
	// which job is marked as received first unless the caller takes additional steps to order the calls.
	// AddJob is non-blocking. If the caller wants to know when their job has finished executing,
	// they need to put that logic into the execute func they pass (which will need to be a closure).
	AddJob(metadata interface{}, conflictsWith func(otherJobMetadata interface{}) bool, execute func()) error

	// Stop shuts down the processor, terminating all goroutines.
	// Incomplete jobs are abandoned may or may not run to completion.
	Stop()

	// GracefulStop shuts down the processor gracefully.
	// After GracefulStop returns, the processor will not accept any new jobs.
	// However, any jobs that are in progress (ie, for which AddJob returned a `nil` error)
	// are allowed to run to completion.
	GracefulStop()

	// Stopped returns whether the processor has been shut down.
	// If Stopped returns true, that means all goroutines spawned by the processor
	// have been shut down.
	Stopped() bool
}

// A dagNode is a node in the DAG representing the jobs to be run.
// Each node stores both child and parent edges, so that the DAG can be efficiently
// traversed in both directions.
type dagNode struct {
	// execute MUST NOT be modified by the processLoop since it is accessed by the worker goroutines.
	// All other fields can be freely accessed by the processLoop, and must NOT be accessed in any way
	// by the worker goroutines.

	metadata interface{}
	execute  func()

	numBlocking int
	blocks      map[*dagNode]struct{}
}

// The dag represents the directed acyclic graph of jobs that have not yet been completed.
// It stores a reference to all the nodes that are unblocked (and ready to execute).
// Since it is a DAG, it is guaranteed that, unless the DAG is empty,
// there will always be at least one unblocked node.
type dag struct {
	allNodes       map[*dagNode]struct{}
	unblockedNodes list.List
}

func newDAG() dag {
	return dag{
		allNodes: make(map[*dagNode]struct{}),
	}
}

// addJob adds a new job to the DAG, adding dependencies by invoking the conflictsWith function.
func (d *dag) addJob(metadata interface{}, isBlockedBy func(interface{}) bool, execute func()) {
	newNode := &dagNode{
		metadata: metadata,
		execute:  execute,
		blocks:   make(map[*dagNode]struct{}),
	}
	if isBlockedBy != nil {
		for existingNode := range d.allNodes {
			if isBlockedBy(existingNode.metadata) {
				newNode.numBlocking++
				existingNode.blocks[newNode] = struct{}{}
			}
		}
	}
	d.allNodes[newNode] = struct{}{}
	if newNode.numBlocking == 0 {
		d.unblockedNodes.PushBack(newNode)
	}
}

// removeJob removes a job from the DAG. All jobs that it blocks lose this edges,
// and if this makes them unblocked, they are moved to the unblockedNodes set.
// It is assumed that the removed job is in the unblocked nodes, since other
// jobs cannot be removed.
func (d *dag) removeJob(node *dagNode) {
	for blockedNode := range node.blocks {
		blockedNode.numBlocking--
		if blockedNode.numBlocking == 0 {
			d.unblockedNodes.PushBack(blockedNode)
		}
	}
	delete(d.allNodes, node)
}

// popUnblockedJob pops an unblocked job from the list
// or returns nil if there are no unblocked jobs.
func (d *dag) popUnblockedJob() *dagNode {
	front := d.unblockedNodes.Front()
	if front == nil {
		return nil
	}
	d.unblockedNodes.Remove(front)
	return front.Value.(*dagNode)
}

type jobRequest struct {
	metadata      interface{}
	execute       func()
	conflictsWith func(interface{}) bool
}

type jobProcessorImpl struct {
	jobDAG          dag
	stopSig         Signal
	gracefulStopSig Signal

	maxWorkers            int
	currentRunningWorkers int

	jobReqC        chan jobRequest
	completedJobsC chan *dagNode
}

func (j *jobProcessorImpl) AddJob(metadata interface{}, conflictsWith func(otherJobMetadata interface{}) bool, execute func()) error {
	// We check this first to avoid the chance that AddJob succeeds after a call to GracefulStop().
	// (Since the process loop may be in its select statement, it is conceivable that the job request
	// case succeeds instead of the gracefulStopSig case).
	if j.gracefulStopSig.IsDone() {
		return ErrJobProcessorStopped
	}
	select {
	case j.jobReqC <- jobRequest{metadata: metadata, conflictsWith: conflictsWith, execute: execute}:
	case <-j.gracefulStopSig.Done():
		return ErrJobProcessorStopped
	case <-j.stopSig.Done():
		return ErrJobProcessorStopped
	}
	return nil
}

func (j *jobProcessorImpl) Stop() {
	j.stopSig.Signal()
}

func (j *jobProcessorImpl) Stopped() bool {
	return j.stopSig.IsDone()
}

func (j *jobProcessorImpl) GracefulStop() {
	j.gracefulStopSig.Signal()
}

func (j *jobProcessorImpl) sendReadyJobsToAvailableWorkers() {
	for j.currentRunningWorkers < j.maxWorkers {
		nextJob := j.jobDAG.popUnblockedJob()
		if nextJob == nil {
			return
		}
		j.currentRunningWorkers++
		go j.runJob(nextJob)
	}
}

// The processLoop does all the meta-processing for the processor (that is, everything except the execution of the
// jobs itself). It helps sequentialize all data access to the processor's data structures, thus avoiding the need
// for locks.
func (j *jobProcessorImpl) processLoop() {
	for !j.stopSig.IsDone() {
		var gracefulStopWhenIdle <-chan struct{}
		if j.currentRunningWorkers == 0 {
			// Only select on j.gracefulStopSig.Done() when there are no running workers, because otherwise, we
			// will want to process the completed job before stopping the job processor (and we know that
			// this select won't block forever since there is going to be a completed job eventually).
			gracefulStopWhenIdle = j.gracefulStopSig.Done()
		}
		select {
		case jobReq := <-j.jobReqC:
			j.jobDAG.addJob(jobReq.metadata, jobReq.conflictsWith, jobReq.execute)
		case completedJobNode := <-j.completedJobsC:
			j.currentRunningWorkers--
			j.jobDAG.removeJob(completedJobNode)
		case <-j.stopSig.Done():
			return
		case <-gracefulStopWhenIdle:
			// Intentionally don't return here, since we do still want to signal j.stopSig.
			// We _could_ just put the following here:
			//
			// j.stopSig.Signal()
			// return
			//
			// But leaving it blank and allowing it to happen through the code below feels more DRY.
		case <-j.stopSig.Done():
			return
		}
		j.sendReadyJobsToAvailableWorkers()
		if j.gracefulStopSig.IsDone() && j.currentRunningWorkers == 0 {
			j.stopSig.Signal()
		}
	}
}

func (j *jobProcessorImpl) runJob(job *dagNode) {
	job.execute()
	select {
	case j.completedJobsC <- job:
	case <-j.stopSig.Done():
	}
}

// NewJobProcessor returns a new, ready-to-use JobProcessor, and kicks off a number of worker goroutines.
// equal to numWorkers, as well as the job processor's main processing loop.
// See the comments on the JobProcessor interface for details on how to use this.
// If numWorkers is <= 0, a number of workers equal to the number of CPUs
// is used.
func NewJobProcessor(numWorkers int) JobProcessor {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	j := &jobProcessorImpl{
		jobDAG:          newDAG(),
		stopSig:         NewSignal(),
		gracefulStopSig: NewSignal(),

		maxWorkers: numWorkers,

		jobReqC:        make(chan jobRequest),
		completedJobsC: make(chan *dagNode),
	}

	go j.processLoop()
	return j
}
