package ops

// Op represents a bolt operation that we want to time.
//go:generate stringer -type=Op
type Op int

// The following is the list of Bolt operations that we want to time.
const (
	Add Op = iota
	Count
	Get
	GetAll
	GetMany
	List
	Rename
	Remove
	Update
	Upsert
	UpsertAll
)
