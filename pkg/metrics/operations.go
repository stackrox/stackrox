package metrics

// Op represents a bolt operation that we want to time.
//
//go:generate stringer -type=Op
type Op int

// The following is the list of Bolt operations that we want to time.
const (
	Add Op = iota
	AddMany

	Count

	Dedupe

	Exists

	Get
	GetAll
	GetMany
	GetFlowsForDeployment
	GetByQuery

	// Special operation currently used only for processes.
	GetGrouped

	// Special operation used for ProcessListeningOnPort
	GetProcessListeningOnPort

	List

	Prune

	Reset
	Rename
	Remove
	RemoveMany
	RemoveFlowsByDeployment

	Search
	Sync

	Update
	UpdateMany
	Upsert
	UpsertAll

	Walk
	WalkByQuery
)
