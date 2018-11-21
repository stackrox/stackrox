package processindicator

// IDAndArgs has the id and args of a process indicator.
type IDAndArgs struct {
	ID   string
	Args string
}

// ProcessWithContainerInfo has information that uniquely identifies a process name.
type ProcessWithContainerInfo struct {
	PodID         string
	ContainerName string
	ProcessName   string
}
