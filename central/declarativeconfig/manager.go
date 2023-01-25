package declarativeconfig

// Manager manages reconciling declarative configurations.
type Manager interface {
	WatchDeclarativeConfigDir()
}
