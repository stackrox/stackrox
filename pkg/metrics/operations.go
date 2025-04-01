package metrics

// Op represents a database operation that we want to time.
//
//go:generate stringer -type=Op
type Op int

// The following is the list of database operations that we want to time.
const (
	Add Op = iota
	AddMany

	Count

	Dedupe

	Exists

	Get
	GetAll
	GetMany
	GetExternalFlowsForDeployment
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

	Unset

	Dropped
)
