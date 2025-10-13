package env

var (
	// ProcessesListeningOnPort enables the NetworkFlow code to also update the processes that are listening on ports
	ProcessesListeningOnPort = RegisterBooleanSetting("ROX_PROCESSES_LISTENING_ON_PORT", true)
)
